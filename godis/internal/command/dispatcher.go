package command

import (
	"strings"

	"github.com/julianschreiner/godis/internal/protocol"
)

type Dispatcher struct {
	encoder protocol.Encoder
}

func NewDispatcher(encoder protocol.Encoder) *Dispatcher {
	return &Dispatcher{
		encoder: encoder,
	}
}

func (d *Dispatcher) Execute(args [][]byte) []byte {
	if len(args) == 0 {
		return d.encoder.EncodeError(ErrEmptyCommand)
	}

	cmd := GodisCommand(strings.ToUpper(string(args[0])))

	switch cmd {
	case CommandPing:
		return d.handlePing(args[1:])
	case CommandEcho:
		return d.handleEcho(args[1:])
	default:
		return d.encoder.EncodeError(ErrUnkownCommand)
	}
}

func (d *Dispatcher) handlePing(args [][]byte) []byte {
	switch len(args) {
	case 0:
		return d.encoder.EncodeSimpleString("PONG")
	case 1:
		return d.encoder.EncodeBulkString(args[0])
	default:
		return d.encoder.EncodeError(
			"ERR wrong number of arguments for 'ping' command",
		)
	}
}

func (d *Dispatcher) handleEcho(args [][]byte) []byte {
	switch len(args) {
	case 1:
		return d.encoder.EncodeBulkString(args[0])
	default:
		return d.encoder.EncodeError(
			"ERR wrong number of arguments for 'echo' command",
		)
	}
}
