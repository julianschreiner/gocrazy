package integration

import (
	"bytes"
	"io"
	"net"
	"testing"
	"time"
)

func TestPipelinedCommands(t *testing.T) {
	conn, err := net.Dial("tcp", "127.0.0.1:6380")
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	conn.SetDeadline(time.Now().Add(2 * time.Second))

	// Two commands, sent using one Write call.
	commands := []byte(
		"*1\r\n$4\r\nPING\r\n" +
			"*2\r\n$4\r\nECHO\r\n$5\r\nhello\r\n",
	)

	n, err := conn.Write(commands)
	if err != nil {
		t.Fatal(err)
	}
	if n != len(commands) {
		t.Fatalf("wrote %d of %d bytes", n, len(commands))
	}

	// Expect two ordered RESP2 responses.
	expected := []byte("+PONG\r\n$5\r\nhello\r\n")
	actual := make([]byte, len(expected))

	if _, err := io.ReadFull(conn, actual); err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(actual, expected) {
		t.Fatalf("expected %q, got %q", expected, actual)
	}
}
