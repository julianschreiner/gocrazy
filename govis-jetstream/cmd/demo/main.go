package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/julianschreiner/govis-jetstream/internal/demo"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

func main() {
	log.SetFlags(log.Ltime | log.Lmicroseconds)
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	if len(os.Args) < 2 {
		return fmt.Errorf("usage: demo setup|publisher|worker <email|report|image>")
	}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	url := env("NATS_URL", nats.DefaultURL)
	var nc *nats.Conn
	var err error
	for ctx.Err() == nil {
		nc, err = nats.Connect(url, nats.Name("jetstream-demo-"+os.Args[1]), nats.Timeout(2*time.Second), nats.DrainTimeout(5*time.Second))
		if err == nil {
			break
		}
		log.Printf("waiting for NATS: %v", err)
		select {
		case <-ctx.Done():
			return nil
		case <-time.After(time.Second):
		}
	}
	if nc == nil {
		return nil
	}
	defer nc.Close()
	defer func() {
		if err := nc.Drain(); err != nil && ctx.Err() == nil {
			log.Printf("drain connection: %v", err)
		}
	}()
	js, err := jetstream.New(nc)
	if err != nil {
		return err
	}
	switch os.Args[1] {
	case "setup":
		setupCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
		defer cancel()
		if err := demo.Setup(setupCtx, js); err != nil {
			return err
		}
		log.Print("streams JOBS and EVENTS with durable consumers ready")
		return nil
	case "publisher":
		return publish(ctx, js)
	case "worker":
		if len(os.Args) != 3 {
			return fmt.Errorf("worker requires email, report, or image")
		}
		return work(ctx, js, os.Args[2])
	default:
		return fmt.Errorf("unknown command %q", os.Args[1])
	}
}

func publish(ctx context.Context, js jetstream.JetStream) error {
	rate, err := strconv.ParseFloat(env("PUBLISH_RATE", "4"), 64)
	if err != nil || rate <= 0 || rate > 1000 {
		return fmt.Errorf("PUBLISH_RATE must be > 0 and <= 1000 jobs/s")
	}
	burstSize, err := strconv.Atoi(env("BURST_SIZE", "0"))
	if err != nil || burstSize < 0 || burstSize > 10000 {
		return fmt.Errorf("BURST_SIZE must be between 0 and 10000")
	}
	burstEvery, err := time.ParseDuration(env("BURST_EVERY", "10s"))
	if err != nil || burstEvery <= 0 {
		return fmt.Errorf("BURST_EVERY must be a positive duration")
	}
	regular := time.NewTicker(time.Duration(float64(time.Second) / rate))
	defer regular.Stop()
	var burst *time.Ticker
	var bursts <-chan time.Time
	if burstSize > 0 {
		burst = time.NewTicker(burstEvery)
		defer burst.Stop()
		bursts = burst.C
	}
	log.Printf("publisher rate=%.2f jobs/s burst=%d every=%s", rate, burstSize, burstEvery)
	runID := strconv.FormatInt(time.Now().UnixNano(), 36)
	var seq uint64
	for {
		count := 0
		select {
		case <-ctx.Done():
			return nil
		case <-regular.C:
			count = 1
		case <-bursts:
			count = burstSize
		}
		for i := 0; i < count && ctx.Err() == nil; i++ {
			seq++
			job := demo.NewJob(runID, seq)
			body, _ := json.Marshal(job)
			msg := &nats.Msg{Subject: "jobs." + job.Kind, Data: body, Header: nats.Header{}}
			msg.Header.Set("Nats-Msg-Id", job.ID)
			msg.Header.Set("Job-Id", job.ID)
			msg.Header.Set("Job-Outcome", job.Outcome)
			pubCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
			ack, err := js.PublishMsg(pubCtx, msg)
			cancel()
			if err != nil {
				if ctx.Err() != nil {
					return nil
				}
				log.Printf("publish %s failed: %v", job.ID, err)
				continue
			}
			if seq <= 3 || seq%25 == 0 {
				log.Printf("stored job=%s kind=%s outcome=%s stream_seq=%d", job.ID, job.Kind, job.Outcome, ack.Sequence)
			}
			if seq%4 == 0 {
				event := &nats.Msg{Subject: "events." + job.Kind, Data: body, Header: nats.Header{}}
				event.Header.Set("Nats-Msg-Id", "event-"+job.ID)
				event.Header.Set("Job-Id", job.ID)
				eventCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
				_, err := js.PublishMsg(eventCtx, event)
				cancel()
				if err != nil && ctx.Err() == nil {
					log.Printf("publish event for %s failed: %v", job.ID, err)
				}
			}
		}
	}
}

func work(ctx context.Context, js jetstream.JetStream, kind string) error {
	name, err := demo.ConsumerName(kind)
	if err != nil {
		return err
	}
	consumer, err := js.Consumer(ctx, demo.Stream, name)
	if err != nil {
		return err
	}
	delay, err := time.ParseDuration(env("PROCESS_DELAY", "150ms"))
	if err != nil || delay < 0 || delay >= 10*time.Second {
		return fmt.Errorf("PROCESS_DELAY must be >= 0 and < 10s")
	}
	log.Printf("worker kind=%s consumer=%s delay=%s", kind, name, delay)
	for ctx.Err() == nil {
		batch, err := consumer.Fetch(1, jetstream.FetchMaxWait(time.Second))
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			log.Printf("fetch: %v", err)
			continue
		}
		for msg := range batch.Messages() {
			if !wait(ctx, delay) {
				return nil // leave the in-flight job for redelivery
			}
			job, decodeErr := demo.Decode(msg.Data())
			meta, metaErr := msg.Metadata()
			if metaErr != nil {
				return metaErr
			}
			var action string
			switch {
			case decodeErr != nil:
				action, err = "TERM-invalid", msg.Term()
			case job.Outcome == "term":
				action, err = "TERM", msg.Term()
			case job.Outcome == "nak-once" && meta.NumDelivered == 1:
				action, err = "NAK", msg.NakWithDelay(350*time.Millisecond)
			default:
				action, err = "ACK", msg.Ack()
			}
			if err != nil {
				log.Printf("%s job=%s attempt=%d failed: %v", action, job.ID, meta.NumDelivered, err)
				continue
			}
			log.Printf("%s job=%s stream_seq=%d attempt=%d", action, job.ID, meta.Sequence.Stream, meta.NumDelivered)
		}
		if err := batch.Error(); err != nil && !errors.Is(err, jetstream.ErrMsgIteratorClosed) {
			log.Printf("fetch batch: %v", err)
		}
	}
	return nil
}

func wait(ctx context.Context, d time.Duration) bool {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
