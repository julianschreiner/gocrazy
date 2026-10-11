package integration

import (
	"bytes"
	"errors"
	"io"
	"net"
	"testing"
	"time"
)

func TestFragmentedCommand(t *testing.T) {
	conn, err := net.Dial("tcp", "127.0.0.1:6380")
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	firstPart := []byte("*2\r\n$4\r\nECHO\r\n$5\r\nhe")
	secondPart := []byte("llo\r\n")

	// 1. Send an incomplete RESP2 command.
	if _, err := conn.Write(firstPart); err != nil {
		t.Fatal(err)
	}

	// 2. The server should not respond yet.
	if err := conn.SetReadDeadline(time.Now().Add(150 * time.Millisecond)); err != nil {
		t.Fatal(err)
	}

	buffer := make([]byte, 1)
	n, err := conn.Read(buffer)

	if n > 0 {
		t.Fatal("server responded before command was complete")
	}

	var netErr net.Error
	if !errors.As(err, &netErr) || !netErr.Timeout() {
		t.Fatalf("expected read timeout, got %v", err)
	}

	// 3. Send the remaining bytes.
	if _, err := conn.Write(secondPart); err != nil {
		t.Fatal(err)
	}

	// 4. Allow time for the server to respond.
	if err := conn.SetReadDeadline(time.Now().Add(2 * time.Second)); err != nil {
		t.Fatal(err)
	}

	expected := []byte("$5\r\nhello\r\n")
	actual := make([]byte, len(expected))

	if _, err := io.ReadFull(conn, actual); err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(actual, expected) {
		t.Fatalf("expected %q, got %q", expected, actual)
	}
}
