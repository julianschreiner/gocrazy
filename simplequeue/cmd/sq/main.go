package main

import (
	"errors"
	"fmt"
	"sync"
	"time"
)

var ErrQueueClosed = errors.New("queue closed")

type Queue struct {
	name   string
	ch     chan string
	closed bool
}

type Orchestrator struct {
	queues map[string]*Queue
	mu     sync.RWMutex
}

func New() *Orchestrator {
	return &Orchestrator{
		queues: make(map[string]*Queue),
	}
}

func (o *Orchestrator) CreateQueue(name string, buffer int) {
	o.mu.Lock()
	defer o.mu.Unlock()

	_, ok := o.queues[name]
	if !ok {
		o.queues[name] = &Queue{
			name: name,
			ch:   make(chan string, buffer),
		}
	}
}

func (o *Orchestrator) GetQueue(name string) (*Queue, bool) {
	o.mu.RLock()
	defer o.mu.RUnlock()

	mq, ok := o.queues[name]
	return mq, ok
}

func (o *Orchestrator) CloseQueue(name string) error {
	o.mu.Lock()
	defer o.mu.Unlock()
	mq, ok := o.queues[name]
	if !ok {
		return errors.New("queue not found")
	}
	if mq.closed {
		return errors.New("queue already closed")
	}

	close(mq.ch)
	mq.closed = true
	return nil
}

func (o *Orchestrator) Publish(name string, msg string) error {
	o.mu.RLock()
	mq, ok := o.queues[name]
	defer o.mu.RUnlock()
	if !ok {
		return errors.New("queue not found")
	}
	if mq.closed {
		return errors.New("queue already closed")
	}

	mq.ch <- msg

	return nil
}

func (o *Orchestrator) Consume(name string) (string, error) {
	o.mu.RLock()
	mq, ok := o.queues[name]
	defer o.mu.RUnlock()
	if !ok {
		return "", errors.New("queue not found")
	}

	msg, ok := <-mq.ch
	if !ok {
		return "", ErrQueueClosed
	}

	return msg, nil
}

func (o *Orchestrator) DeleteQueue(name string) error {
	o.mu.Lock()
	defer o.mu.Unlock()

	mq, ok := o.queues[name]
	if !ok {
		return errors.New("queue not found")
	}

	if !mq.closed {
		close(mq.ch)
		mq.closed = true
	}

	delete(o.queues, name)
	return nil
}

func main() {
	orchestrator := New()
	orchestrator.CreateQueue("log", 100)
	orchestrator.CreateQueue("jobs", 0) // worker handoff, if no worker is available the producer waits

	logMessages := []string{"application:started", "worker:ready"}

	go func() {
		println("goroutine: log publisher: sending job")
		for _, message := range logMessages {
			if err := orchestrator.Publish("log", message); err != nil {
				fmt.Println("log publisher error:", err)
				return
			}
			fmt.Println("log publisher sent:", message)
		}
	}()

	fmt.Println("main: consuming log")
	for range logMessages {
		logMessage, err := orchestrator.Consume("log")
		if err != nil {
			panic(err)
		}
		fmt.Println("log consumer received:", logMessage)
	}
	fmt.Println("main: deleting log")
	if err := orchestrator.DeleteQueue("log"); err != nil {
		panic(err)
	}

	go func() {
		fmt.Println("goroutine: job publisher: sending job")
		if err := orchestrator.Publish("jobs", "process:order"); err != nil {
			fmt.Println("job publisher error:", err)
			return
		}
		fmt.Println("job publisher: send finished")
	}()

	fmt.Println("main: consuming job")
	job, err := orchestrator.Consume("jobs")
	if err != nil {
		panic(err)
	}
	fmt.Println("main: received job:", job)

	fmt.Println("main: deleting jobs")
	if err := orchestrator.DeleteQueue("jobs"); err != nil {
		panic(err)
	}

	fmt.Println("\nmain: starting event stream")
	orchestrator.CreateQueue("events", 0)

	var eventPublishers sync.WaitGroup
	for publisherID := 1; publisherID <= 3; publisherID++ {
		eventPublishers.Add(1)
		go func(id int) {
			defer eventPublishers.Done()
			for eventNumber := 1; eventNumber <= id; eventNumber++ {
				message := fmt.Sprintf("publisher %d event %d", id, eventNumber)
				if err := orchestrator.Publish("events", message); err != nil {
					fmt.Println("event publisher error:", err)
					return
				}
			}
		}(publisherID)
	}

	var eventConsumers sync.WaitGroup
	eventConsumers.Add(1)
	go func() {
		defer eventConsumers.Done()
		for {
			message, err := orchestrator.Consume("events")
			if err == ErrQueueClosed {
				break
			}
			if err != nil {
				panic(err)
			}
			fmt.Println("event consumer received:", message)
		}
	}()

	eventPublishers.Wait()
	if err := orchestrator.CloseQueue("events"); err != nil {
		panic(err)
	}

	eventConsumers.Wait()

	fmt.Println("main: all publishers finished and all events received")
	if err := orchestrator.DeleteQueue("events"); err != nil {
		panic(err)
	}

	fmt.Println("\nmain: starting background save")
	var save sync.WaitGroup
	save.Add(1)
	go func() {
		defer save.Done()
		fmt.Println("save: working")
		time.Sleep(300 * time.Millisecond) // Simulate slow io
		fmt.Println("save: finished")
	}()

	fmt.Println("main: waiting for save before exit")
	save.Wait()
	fmt.Println("main: save complete; exiting")
}
