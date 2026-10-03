package main

import (
	"fmt"
	"time"

	"github.com/julianschreiner/roundrobin/internal/process"
)

func main() {
	// register processes so we can schedule them
	queue := initQueue()
	runScheduler(queue)

	for _, p := range queue.Processes {
		fmt.Printf(
			"ProcessID: P%d used %v of Turnaround time, with a burst time of: %v\n",
			p.ProcessID,
			p.TurnAroundTime,
			p.BurstTime(),
		)
	}
}

func initQueue() *process.Queue {
	return &process.Queue{
		Processes: []*process.Process{
			process.CreateProcess(1, 5*time.Millisecond),
			process.CreateProcess(2, 4*time.Millisecond),
			process.CreateProcess(3, 2*time.Millisecond),
			process.CreateProcess(4, 1*time.Millisecond),
			process.CreateProcess(5, 10*time.Millisecond),
			process.CreateProcess(6, 11*time.Millisecond),
			process.CreateProcess(7, 14*time.Millisecond),
			process.CreateProcess(8, 19*time.Millisecond),
			process.CreateProcess(9, 288*time.Millisecond),
			process.CreateProcess(10, 7*time.Millisecond),
		},
		Quantum: 2 * time.Millisecond,
	}
}

func runScheduler(pq *process.Queue) {
	var finishedProcesses int
	start := time.Now()
	// bonus later: print current order of processes
	// bonus later: shift processes at the end of the slice when they finished their turn
	// bonus later: remove processes when they finished their burst
	for len(pq.Processes) > finishedProcesses {
		for _, p := range pq.Processes {
			if p.State == process.Finished {
				continue
			}
			p.State = process.Running
			p.RemainingBurstTime = p.RemainingBurstTime - pq.Quantum
			if p.RemainingBurstTime <= 0 {
				time.Sleep(pq.Quantum - (-p.RemainingBurstTime)) // simulate part of quantum
				p.State = process.Finished
				p.RemainingBurstTime = 0
				p.TurnAroundTime = time.Since(start)
				finishedProcesses++
			} else {
				time.Sleep(pq.Quantum) // simulate full quantum
				p.State = process.Ready
			}
		}
	}
}
