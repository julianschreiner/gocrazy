package protocol

import (
	"bytes"
	"errors"
	"fmt"
	"reflect"
	"testing"
)

func TestParsePing(t *testing.T) {
	respParser := NewResp2Parser()
	data := []byte("*1\r\n$4\r\nPING\r\n")

	args, bytesConsumed, err := respParser.ParseCommand(data)
	if err != nil {
		t.Fatal(err)
	}

	if bytesConsumed != 14 {
		t.Fatalf("expected 14 bytes consumed, got %d", bytesConsumed)
	}

	if len(args) != 1 {
		t.Fatalf("expected 1 argument, got %d", len(args))
	}

	if !bytes.Equal(args[0], []byte("PING")) {
		t.Fatalf("expected PING, got %q", args[0])
	}
}

func TestParseSetCommand(t *testing.T) {
	parser := NewResp2Parser()
	data := []byte("*3\r\n$3\r\nSET\r\n$4\r\nuser\r\n$1\r\n1\r\n")

	args, consumed, err := parser.ParseCommand(data)
	if err != nil {
		t.Fatal(err)
	}

	expected := [][]byte{
		[]byte("SET"),
		[]byte("user"),
		[]byte("1"),
	}

	if !reflect.DeepEqual(args, expected) {
		t.Fatalf("expected %q, got %q", expected, args)
	}

	if consumed != len(data) {
		t.Fatalf("expected %d bytes consumed, got %d", len(data), consumed)
	}
}

func TestParseIncompleteCommand(t *testing.T) {
	tests := []struct {
		name string
		data string
	}{
		{"empty input", ""},
		{"incomplete array header", "*1"},
		{"incomplete bulk header", "*1\r\n$4"},
		{"incomplete bulk data", "*1\r\n$4\r\nPI"},
		{"incomplete trailing CRLF", "*1\r\n$4\r\nPING\r"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := NewResp2Parser()

			_, consumed, err := parser.ParseCommand([]byte(tt.data))

			if !errors.Is(err, ErrIncomplete) {
				t.Fatalf("expected ErrIncomplete, got %v", err)
			}

			if consumed != 0 {
				t.Fatalf("expected 0 bytes consumed, got %d", consumed)
			}
		})
	}
}

func TestParsePipelinedCommands(t *testing.T) {
	parser := NewResp2Parser()

	first := []byte("*1\r\n$4\r\nPING\r\n")
	second := []byte("*2\r\n$4\r\nECHO\r\n$5\r\nhello\r\n")

	data := append(append([]byte{}, first...), second...)

	// Parse the first command.
	args1, consumed1, err := parser.ParseCommand(data)
	if err != nil {
		t.Fatal(err)
	}

	expected1 := [][]byte{[]byte("PING")}

	if !reflect.DeepEqual(args1, expected1) {
		t.Fatalf("expected %q, got %q", expected1, args1)
	}

	if consumed1 != len(first) {
		t.Fatalf("expected %d bytes consumed, got %d", len(first), consumed1)
	}

	// Parse the second command from the remaining bytes.
	args2, consumed2, err := parser.ParseCommand(data[consumed1:])
	if err != nil {
		t.Fatal(err)
	}

	expected2 := [][]byte{
		[]byte("ECHO"),
		[]byte("hello"),
	}

	if !reflect.DeepEqual(args2, expected2) {
		t.Fatalf("expected %q, got %q", expected2, args2)
	}

	if consumed1+consumed2 != len(data) {
		t.Fatalf("expected all %d bytes consumed, got %d",
			len(data), consumed1+consumed2)
	}
}

func TestParseMalformedCommand(t *testing.T) {
	tests := []struct {
		name string
		data string
	}{
		{"invalid array prefix", "+PING\r\n"},
		{"invalid array count", "*abc\r\n"},
		{"negative array count", "*-1\r\n"},
		{"empty command array", "*0\r\n"},
		{"invalid bulk prefix", "*1\r\n+PING\r\n"},
		{"invalid bulk length", "*1\r\n$abc\r\n"},
		{"negative bulk length", "*1\r\n$-5\r\n"},
		{"invalid trailing CRLF", "*1\r\n$4\r\nPINGxx"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := NewResp2Parser()

			_, consumed, err := parser.ParseCommand([]byte(tt.data))

			if !errors.Is(err, ErrMalformed) {
				t.Fatalf("expected ErrMalformed, got %v", err)
			}

			if consumed != 0 {
				t.Fatalf("expected 0 bytes consumed, got %d", consumed)
			}
		})
	}
}

func TestParseBinarySafeArgument(t *testing.T) {
	parser := NewResp2Parser()

	// Contains a null byte, CRLF, and a space.
	payload := []byte{'A', 0, '\r', '\n', ' ', 'B'}

	data := []byte(fmt.Sprintf(
		"*2\r\n$4\r\nECHO\r\n$%d\r\n",
		len(payload),
	))
	data = append(data, payload...)
	data = append(data, '\r', '\n')

	args, consumed, err := parser.ParseCommand(data)
	if err != nil {
		t.Fatal(err)
	}

	if len(args) != 2 {
		t.Fatalf("expected 2 arguments, got %d", len(args))
	}

	if !bytes.Equal(args[0], []byte("ECHO")) {
		t.Fatalf("expected ECHO, got %q", args[0])
	}

	if !bytes.Equal(args[1], payload) {
		t.Fatalf("expected %v, got %v", payload, args[1])
	}

	if consumed != len(data) {
		t.Fatalf("expected %d bytes consumed, got %d", len(data), consumed)
	}
}

func TestMaxBulkStringLengthExceeded(t *testing.T) {
	parser := NewResp2Parser()

	// Declare a bulk string exceeding the configured limit.
	// No need to allocate the actual payload.
	data := []byte(fmt.Sprintf(
		"*1\r\n$%d\r\n",
		maxBulkStringLength+1,
	))

	_, consumed, err := parser.ParseCommand(data)

	if !errors.Is(err, ErrMalformed) {
		t.Fatalf("expected ErrMalformed, got %v", err)
	}

	if consumed != 0 {
		t.Fatalf("expected 0 bytes consumed, got %d", consumed)
	}
}
