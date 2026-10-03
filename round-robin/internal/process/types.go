package process

import "time"

type ProcessState uint8

const (
	Ready ProcessState = iota
	Running
	Finished
)

type Queue struct {
	Processes []*Process
	Quantum   time.Duration
}

type Process struct {
	ProcessID          uint16
	burstTime          time.Duration
	RemainingBurstTime time.Duration
	State              ProcessState
	TurnAroundTime     time.Duration
}

func CreateProcess(processID uint16, burstTime time.Duration) *Process {
	return &Process{
		ProcessID:          uint16(processID),
		burstTime:          burstTime,
		RemainingBurstTime: burstTime,
		State:              Ready,
	}
}

func (p *Process) BurstTime() time.Duration {
	return p.burstTime
}
