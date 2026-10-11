package eventloop

import (
	"errors"
	"time"

	"golang.org/x/sys/unix"
)

type PollerLinux struct {
	epollFd int
}

func (p *PollerLinux) GetEpollFd() int {
	return p.epollFd
}

func NewPollerLinux() (*PollerLinux, error) {
	ePollFd, err := unix.EpollCreate1(unix.EPOLL_CLOEXEC)
	if err != nil {
		return nil, err
	}
	return &PollerLinux{
		epollFd: ePollFd,
	}, nil
}

func (p *PollerLinux) Register(fd int, interest Interest) error {
	return p.control(
		fd,
		interest,
		unix.EPOLL_CTL_ADD,
	)
}

func (p *PollerLinux) Modify(fd int, interest Interest) error {
	return p.control(
		fd,
		interest,
		unix.EPOLL_CTL_MOD,
	)
}

func (p *PollerLinux) Unregister(fd int) error {
	return unix.EpollCtl(
		p.GetEpollFd(),
		unix.EPOLL_CTL_DEL,
		fd,
		nil,
	)
}

// timeout: -1 block indefinitely
// timeout: 0 return immediately
// timeout: wait up to x
func (p *PollerLinux) Wait(timeout time.Duration) ([]Event, error) {
	events := make([]unix.EpollEvent, 128)

	for {

		n, err := unix.EpollWait(
			p.GetEpollFd(),
			events,
			toMilliseconds(timeout),
		)

		if errors.Is(err, unix.EINTR) {
			continue // Retry epoll_wait
			// later: calculate the remaining timeout using a deadline
		}

		if err != nil {
			return nil, err
		}

		var pollerEvents []Event
		for _, event := range events[:n] {
			pollerEvents = append(pollerEvents, Event{
				FD:       int(event.Fd),
				Readable: event.Events&unix.EPOLLIN != 0,
				Writable: event.Events&unix.EPOLLOUT != 0,
				Error:    event.Events&unix.EPOLLERR != 0,
				Hangup:   event.Events&(unix.EPOLLHUP|unix.EPOLLRDHUP) != 0,
			})
		}

		return pollerEvents, nil
	}
}

func (p *PollerLinux) Close() error {
	return unix.Close(p.GetEpollFd())
}

func toMilliseconds(timeout time.Duration) int {
	if timeout < 0 {
		return -1
	}

	ms := timeout.Milliseconds()
	if timeout > 0 && timeout%time.Millisecond != 0 {
		ms++
	}

	return int(ms)
}

func (p *PollerLinux) control(
	fd int,
	interest Interest,
	operation int,
) error {
	event := unix.EpollEvent{
		Events: toEpollEvents(interest),
		Fd:     int32(fd),
	}

	return unix.EpollCtl(
		p.epollFd,
		operation,
		fd,
		&event,
	)
}

func toEpollEvents(interest Interest) uint32 {
	var events uint32

	if interest&Readable != 0 {
		events |= unix.EPOLLIN
	}

	if interest&Writable != 0 {
		events |= unix.EPOLLOUT
	}

	// Notify us when the peer closes its writing side.
	// events |= unix.EPOLLRDHUP

	return events
}
