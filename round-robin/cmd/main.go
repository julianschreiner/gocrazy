package main

import (
	"fmt"
	"os"
	"slices"
	"strconv"
	"time"

	"github.com/julianschreiner/roundrobin/internal/process"
)

func main() {
	// register processes so we can schedule them
	queue := initQueue()
	if queue.Quantum <= 0 {
		fmt.Println("error queue.Quantum needs to be > 0")
		os.Exit(1)
	}
	runScheduler(queue)

	println("====== FINAL RESULT ======")
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
		Quantum: 3 * time.Millisecond,
	}
}

func runScheduler(pq *process.Queue) {
	start := time.Now()
	copiedProcesses := slices.Clone(pq.Processes)
	for len(copiedProcesses) > 0 {
		for _, p := range copiedProcesses {
			printCurrentOrder(copiedProcesses)
			p.State = process.Running
			p.RemainingBurstTime = p.RemainingBurstTime - pq.Quantum
			if p.RemainingBurstTime <= 0 {
				time.Sleep(pq.Quantum - (-p.RemainingBurstTime)) // simulate part of quantum
				p.State = process.Finished
				p.RemainingBurstTime = 0
				p.TurnAroundTime = time.Since(start)
				// process finished their burst, so remove it
				copiedProcesses = copiedProcesses[1:]
			} else {
				time.Sleep(pq.Quantum) // simulate full quantum
				p.State = process.Ready
				// shift process at the end of the slice, as turn is finished
				cp := p
				copiedProcesses = copiedProcesses[1:]
				copiedProcesses = append(copiedProcesses, cp)
			}
		}
	}
}

func printCurrentOrder(cp []*process.Process) {
	var processIDs []string
	for _, p := range cp {
		processIDs = append(processIDs, fmt.Sprintf("P%v (%v)", strconv.Itoa(int(p.ProcessID)), p.RemainingBurstTime))
	}

	fmt.Printf("Queue: %v\n", processIDs)
}
