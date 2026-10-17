package command

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/rlat3d/godis/internal/persistence"
	"github.com/rlat3d/godis/internal/protocol"
	"github.com/rlat3d/godis/internal/storage"
)

type Dispatcher struct {
	encoder protocol.Encoder
	store   *storage.Store
	aof     *persistence.AOF
}

func NewDispatcher(encoder protocol.Encoder, store *storage.Store, aof *persistence.AOF) *Dispatcher {
	return &Dispatcher{
		encoder: encoder,
		store:   store,
		aof:     aof,
	}
}

// TODO:
// For example, when processing SET user John:
// 1. Dispatcher validates the arguments.
// 2.  Dispatcher appends the mutation to the AOF.
// 3. Dispatcher updates the in-memory store.
// 4. Dispatcher returns +OK.

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
	case CommandKeys:
		return d.handleKeys(args[1:])
	case CommandExpire:
		return d.handleExpire(args[1:])
	case CommandTTL:
		return d.handleTTL(args[1:])
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

func (d *Dispatcher) handleKeys(args [][]byte) []byte {
	switch len(args) {
	case 1:
		var pattern string
		pattern = string(args[0])
		if bytes.Equal(args[0], []byte("*")) {
			pattern = string([]byte("."))
		}

		keys, found, err := d.store.GetKeys(pattern)
		if !found || err != nil {
			return d.encoder.EncodeNullBulkString()
		}
		return d.encoder.EncodeBulkStringArray(keys)
	default:
		return d.encoder.EncodeError(
			fmt.Sprintf(ErrWrongNumberOfArgs, "keys"),
		)
	}
}

func (d *Dispatcher) handleExpire(args [][]byte) []byte {
	switch len(args) {
	case 2:
		dur, err := time.ParseDuration(string(args[1]) + "s")
		if err != nil {
			return d.encoder.EncodeNullBulkString()
		}

		err = d.store.Expire(string(args[0]), dur)
		if err != nil {
			return d.encoder.EncodeNullBulkString()
		}

		return d.encoder.EncodeSimpleString("OK")
	default:
		return d.encoder.EncodeError(
			fmt.Sprintf(ErrWrongNumberOfArgs, "expire"),
		)
	}
}

func (d *Dispatcher) handleTTL(args [][]byte) []byte {
	switch len(args) {
	case 1:
		return d.encoder.EncodeInteger(int64(d.store.TTL(string(args[0]))))
	default:
		return d.encoder.EncodeError(
			fmt.Sprintf(ErrWrongNumberOfArgs, "ttl"),
		)
	}
}
