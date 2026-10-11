package eventloop

import "time"

type Interest uint8

const (
	Readable Interest = 1 << iota
	Writable
)

type Event struct {
	FD       int
	Readable bool
	Writable bool
	Error    bool
	Hangup   bool
}

type Poller interface {
	Register(fd int, interest Interest) error

	Modify(fd int, interest Interest) error

	Unregister(fd int) error

	Wait(timeout time.Duration) ([]Event, error)

	Close() error
}
