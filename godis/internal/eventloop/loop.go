package eventloop

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/julianschreiner/godis/internal/command"
	"github.com/julianschreiner/godis/internal/network"
	"github.com/julianschreiner/godis/internal/protocol"
	"golang.org/x/sys/unix"
)

type Loop struct {
	poller     Poller
	listenerFD int
	clients    map[int]*network.Client
	dispatcher *command.Dispatcher
	parser     protocol.Parser
}

func NewLoop(
	poller Poller,
	listenerFD int,
	dispatcher *command.Dispatcher,
	parser protocol.Parser,
) (*Loop, error) {
	if err := poller.Register(listenerFD, Readable); err != nil {
		return nil, err
	}

	return &Loop{
		poller:     poller,
		listenerFD: listenerFD,
		clients:    make(map[int]*network.Client),
		dispatcher: dispatcher,
		parser:     parser,
	}, nil
}

func (l *Loop) Run(ctx context.Context) error {
	for {
		if ctx.Err() != nil {
			return nil
		}

		events, err := l.poller.Wait(100 * time.Millisecond)
		if err != nil {
			return err
		}

		for _, event := range events {
			if err := l.handleEvent(event); err != nil {
				return err
			}
		}
	}
}

func (l *Loop) handleEvent(event Event) error {
	if event.FD == l.listenerFD {
		if event.Error || event.Hangup {
			return fmt.Errorf("listening socket error")
		}

		if event.Readable {
			return l.acceptConnections()
		}

		return nil
	}

	client, exists := l.clients[event.FD]
	if !exists {
		return nil // Client already removed.
	}

	if event.Readable {
		l.handleRead(event.FD)
	}

	// handleRead() might have removed the client.
	client, exists = l.clients[event.FD]
	if !exists {
		return nil
	}

	if event.Writable {
		l.handleWrite(event.FD)
	}

	// handleWrite() might have removed the client.
	client, exists = l.clients[event.FD]
	if !exists {
		return nil
	}

	if event.Error {
		l.removeClient(event.FD)
		return nil
	}

	if event.Hangup && len(client.WriteBuffer) == 0 {
		l.removeClient(event.FD)
	}

	return nil
}

func (l *Loop) acceptConnections() error {
	for {
		fd, _, err := unix.Accept4(
			l.listenerFD,
			unix.SOCK_NONBLOCK|unix.SOCK_CLOEXEC,
		)

		if err == unix.EAGAIN {
			return nil
		}
		if err == unix.EINTR {
			continue
		}
		if err != nil {
			return err
		}

		if err := l.poller.Register(fd, Readable); err != nil {
			_ = unix.Close(fd)
			return err
		}

		l.clients[fd] = &network.Client{
			FD: fd,
		}
	}
}

func (l *Loop) handleRead(fd int) {
	buffer := make([]byte, 4096)

	n, err := unix.Read(fd, buffer)
	if err == unix.EAGAIN || err == unix.EINTR {
		return
	}
	if err != nil || n == 0 {
		l.removeClient(fd)
		return
	}

	client := l.clients[fd]

	client.ReadBuffer = append(client.ReadBuffer, buffer[:n]...)

	for len(client.ReadBuffer) > 0 {
		args, consumed, err := l.parser.ParseCommand(client.ReadBuffer)

		if errors.Is(err, protocol.ErrIncomplete) {
			break
		}

		if err != nil {
			l.removeClient(fd)
			return
		}

		client.ReadBuffer = client.ReadBuffer[consumed:]

		response := l.dispatcher.Execute(args)

		client.WriteBuffer = append(client.WriteBuffer, response...)
	}

	// Enable writable notifications.
	if err := l.poller.Modify(fd, Readable|Writable); err != nil {
		l.removeClient(fd)
	}
}

func (l *Loop) handleWrite(fd int) {
	client := l.clients[fd]

	/* 	if len(argsWriteBuffer) == 0 {
	return
	}*/

	n, err := unix.Write(fd, client.WriteBuffer)
	if err == unix.EAGAIN {
		return // Try again when writable.
	}
	if err != nil {
		l.removeClient(fd)
		return
	}

	// Remove successfully written bytes.
	client.WriteBuffer = client.WriteBuffer[n:]

	// Stop monitoring writable events when done.
	if len(client.WriteBuffer) == 0 {
		if err := l.poller.Modify(fd, Readable); err != nil {
			l.removeClient(fd)
		}
	}
}

func (l *Loop) removeClient(fd int) {
	_ = l.poller.Unregister(fd)

	_ = unix.Close(fd)

	delete(l.clients, fd)
}

func (l *Loop) Stop(ctx context.Context) {
	// todo graceful shutdown
}
