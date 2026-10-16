package command

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/julianschreiner/godis/internal/protocol"
	"github.com/julianschreiner/godis/internal/storage"
)

type Dispatcher struct {
	encoder protocol.Encoder
	store   *storage.Store
}

func NewDispatcher(encoder protocol.Encoder, store *storage.Store) *Dispatcher {
	return &Dispatcher{
		encoder: encoder,
		store:   store,
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
	case CommandSet:
		return d.handleSet(args[1:])
	case CommandGet:
		return d.handleGet(args[1:])
	case CommandDel:
		return d.handleDel(args[1:])
	case CommandExists:
		return d.handleExists(args[1:])
	case CommandIncr:
		return d.handleIncr(args[1:])
	default:
		return d.encoder.EncodeError(ErrUnknownCommand)
	}
}

func (d *Dispatcher) CleanupExpired() {
	d.store.CleanupExpired()
}

func (d *Dispatcher) handleIncr(args [][]byte) []byte {
	switch len(args) {
	case 1:
		newVal, err := d.store.Incr(string(args[0]))
		if err != nil {
			return d.encoder.EncodeError(ErrValueNoIntegerOrOutOfRange)
		}
		return d.encoder.EncodeInteger(int64(newVal))
	default:
		return d.encoder.EncodeError(
			fmt.Sprintf(ErrWrongNumberOfArgs, "incr"),
		)
	}
}

func (d *Dispatcher) handleExists(args [][]byte) []byte {
	if len(args) >= 1 {
		existsCount := 0
		for _, arg := range args {
			exists := d.store.Exists(string(arg))
			if exists {
				existsCount++
			}
		}

		return d.encoder.EncodeInteger(int64(existsCount))
	}

	return d.encoder.EncodeError(
		fmt.Sprintf(ErrWrongNumberOfArgs, "exists"),
	)
}

func (d *Dispatcher) handleDel(args [][]byte) []byte {
	if len(args) >= 1 {
		delCount := 0
		for _, arg := range args {
			success := d.store.Delete(string(arg))
			if success {
				delCount++
			}
		}

		return d.encoder.EncodeInteger(int64(delCount))
	}

	return d.encoder.EncodeError(
		fmt.Sprintf(ErrWrongNumberOfArgs, "del"),
	)
}

func (d *Dispatcher) handleGet(args [][]byte) []byte {
	switch len(args) {
	case 1:
		val, found := d.store.Get(string(args[0]))
		if !found {
			return d.encoder.EncodeNullBulkString()
		}
		return d.encoder.EncodeBulkString(val)
	default:
		return d.encoder.EncodeError(
			fmt.Sprintf(ErrWrongNumberOfArgs, "get"),
		)
	}
}

func (d *Dispatcher) handleSet(args [][]byte) []byte {
	switch len(args) {
	case 2:
		d.store.Set(string(args[0]), args[1], nil)
		return d.encoder.EncodeSimpleString("OK")
	case 4:
		// SET key value EX seconds
		if !strings.EqualFold(string(args[2]), "EX") {
			return d.encoder.EncodeError("ERR syntax error")
		}

		seconds, err := strconv.ParseInt(string(args[3]), 10, 64)
		if err != nil || seconds <= 0 {
			return d.encoder.EncodeError("ERR invalid expire time in 'set' command")
		}

		expiration := time.Now().Add(time.Duration(seconds) * time.Second)
		expiresAt := &expiration

		d.store.Set(string(args[0]), args[1], expiresAt)
		return d.encoder.EncodeSimpleString("OK")
	default:
		return d.encoder.EncodeError(
			fmt.Sprintf(ErrWrongNumberOfArgs, "set"),
		)
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
			fmt.Sprintf(ErrWrongNumberOfArgs, "ping"),
		)
	}
}

func (d *Dispatcher) handleEcho(args [][]byte) []byte {
	switch len(args) {
	case 1:
		return d.encoder.EncodeBulkString(args[0])
	default:
		return d.encoder.EncodeError(
			fmt.Sprintf(ErrWrongNumberOfArgs, "echo"),
		)
	}
}
