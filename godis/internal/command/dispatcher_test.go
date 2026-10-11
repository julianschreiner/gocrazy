package command

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/julianschreiner/godis/internal/protocol"
	"github.com/julianschreiner/godis/internal/storage"
)

func newTestDispatcher() *Dispatcher {
	return NewDispatcher(
		protocol.NewResp2Encoder(),
		storage.NewStore(),
	)
}

func commandArgs(values ...string) [][]byte {
	args := make([][]byte, len(values))
	for i, value := range values {
		args[i] = []byte(value)
	}
	return args
}

func assertResponse(t *testing.T, got []byte, want string) {
	t.Helper()

	if !bytes.Equal(got, []byte(want)) {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestExecuteBasicCommands(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{"PING", []string{"PING"}, "+PONG\r\n"},
		{"lowercase PING", []string{"ping"}, "+PONG\r\n"},
		{"PING with argument", []string{"PING", "hello"}, "$5\r\nhello\r\n"},
		{"ECHO", []string{"ECHO", "hello"}, "$5\r\nhello\r\n"},
		{"ECHO with spaces", []string{"ECHO", "Hello World"}, "$11\r\nHello World\r\n"},
		{"ECHO empty value", []string{"ECHO", ""}, "$0\r\n\r\n"},
		{"GET missing key", []string{"GET", "missing"}, "$-1\r\n"},
		{"DEL missing key", []string{"DEL", "missing"}, ":0\r\n"},
		{"EXISTS missing key", []string{"EXISTS", "missing"}, ":0\r\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := newTestDispatcher()
			got := d.Execute(commandArgs(tt.args...))
			assertResponse(t, got, tt.want)
		})
	}
}

func TestExecuteStorageCommands(t *testing.T) {
	// All commands share the same store.
	d := newTestDispatcher()

	tests := []struct {
		name string
		args []string
		want string
	}{
		{"SET user", []string{"SET", "user", "John"}, "+OK\r\n"},
		{"GET user", []string{"GET", "user"}, "$4\r\nJohn\r\n"},
		{"overwrite user", []string{"SET", "user", "Jane"}, "+OK\r\n"},
		{"GET updated user", []string{"GET", "user"}, "$4\r\nJane\r\n"},
		{"EXISTS user", []string{"EXISTS", "user"}, ":1\r\n"},
		{"EXISTS duplicates", []string{"EXISTS", "user", "user", "missing"}, ":2\r\n"},
		{"DEL duplicates", []string{"DEL", "user", "user"}, ":1\r\n"},
		{"GET deleted user", []string{"GET", "user"}, "$-1\r\n"},
		{"DEL again", []string{"DEL", "user"}, ":0\r\n"},
		{"INCR missing counter", []string{"INCR", "counter"}, ":1\r\n"},
		{"INCR again", []string{"INCR", "counter"}, ":2\r\n"},
		{"GET counter", []string{"GET", "counter"}, "$1\r\n2\r\n"},
		{"SET counter", []string{"SET", "counter", "41"}, "+OK\r\n"},
		{"INCR existing", []string{"INCR", "counter"}, ":42\r\n"},
		{"GET incremented", []string{"GET", "counter"}, "$2\r\n42\r\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := d.Execute(commandArgs(tt.args...))
			assertResponse(t, got, tt.want)
		})
	}
}

func TestExecuteWrongArgumentCounts(t *testing.T) {
	tests := []struct {
		command string
		args    []string
	}{
		{"ping", []string{"PING", "a", "b"}},
		{"echo", []string{"ECHO"}},
		{"set", []string{"SET", "user"}},
		{"get", []string{"GET"}},
		{"del", []string{"DEL"}},
		{"exists", []string{"EXISTS"}},
		{"incr", []string{"INCR"}},
	}

	for _, tt := range tests {
		t.Run(tt.command, func(t *testing.T) {
			d := newTestDispatcher()

			got := d.Execute(commandArgs(tt.args...))
			want := "-" + fmt.Sprintf(
				ErrWrongNumberOfArgs, tt.command,
			) + "\r\n"

			assertResponse(t, got, want)
		})
	}
}

func TestExecuteUnknownCommand(t *testing.T) {
	d := newTestDispatcher()

	got := d.Execute(commandArgs("UNKNOWN"))
	want := "-" + ErrUnknownCommand + "\r\n"

	assertResponse(t, got, want)
}

func TestExecuteEmptyCommand(t *testing.T) {
	d := newTestDispatcher()

	got := d.Execute(nil)
	want := "-" + ErrEmptyCommand + "\r\n"

	assertResponse(t, got, want)
}

func TestExecuteBinarySafeValues(t *testing.T) {
	d := newTestDispatcher()
	value := []byte{'A', 0, '\r', '\n', 'B'}

	got := d.Execute([][]byte{
		[]byte("SET"),
		[]byte("binary"),
		value,
	})
	assertResponse(t, got, "+OK\r\n")

	got = d.Execute(commandArgs("GET", "binary"))
	want := append([]byte("$5\r\n"), value...)
	want = append(want, '\r', '\n')

	if !bytes.Equal(got, want) {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestExecuteIncrInvalidValues(t *testing.T) {
	tests := []string{
		"hello",
		"12.5",
		"",
		"9223372036854775807", // Increment would overflow.
		"9223372036854775808", // Already out of range.
	}

	for _, value := range tests {
		t.Run(value, func(t *testing.T) {
			d := newTestDispatcher()

			d.Execute(commandArgs("SET", "counter", value))

			got := d.Execute(commandArgs("INCR", "counter"))
			want := "-" + ErrValueNoIntegerOrOutOfRange + "\r\n"

			assertResponse(t, got, want)

			// Failed INCR must leave the stored value unchanged.
			got = d.Execute(commandArgs("GET", "counter"))
			want = fmt.Sprintf("$%d\r\n%s\r\n", len(value), value)

			assertResponse(t, got, want)
		})
	}
}
