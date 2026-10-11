package protocol

import (
	"bytes"
	"testing"
)

func TestEncodeSimpleString(t *testing.T) {
	encoder := NewResp2Encoder()

	got := encoder.EncodeSimpleString("OK")
	want := []byte("+OK\r\n")

	if !bytes.Equal(got, want) {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestEncodeError(t *testing.T) {
	encoder := NewResp2Encoder()

	got := encoder.EncodeError("ERR unknown command")
	want := []byte("-ERR unknown command\r\n")

	if !bytes.Equal(got, want) {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestEncodeInteger(t *testing.T) {
	encoder := NewResp2Encoder()

	tests := []struct {
		name  string
		value int64
		want  string
	}{
		{"positive", 42, ":42\r\n"},
		{"zero", 0, ":0\r\n"},
		{"negative", -42, ":-42\r\n"},
		{"max int64", 9223372036854775807, ":9223372036854775807\r\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := encoder.EncodeInteger(tt.value)

			if !bytes.Equal(got, []byte(tt.want)) {
				t.Fatalf("expected %q, got %q", tt.want, got)
			}
		})
	}
}

func TestEncodeBulkString(t *testing.T) {
	encoder := NewResp2Encoder()

	got := encoder.EncodeBulkString([]byte("hello"))
	want := []byte("$5\r\nhello\r\n")

	if !bytes.Equal(got, want) {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestEncodeEmptyBulkString(t *testing.T) {
	encoder := NewResp2Encoder()

	got := encoder.EncodeBulkString([]byte{})
	want := []byte("$0\r\n\r\n")

	if !bytes.Equal(got, want) {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestEncodeNullBulkString(t *testing.T) {
	encoder := NewResp2Encoder()

	got := encoder.EncodeNullBulkString()
	want := []byte("$-1\r\n")

	if !bytes.Equal(got, want) {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestEncodeBulkStringArray(t *testing.T) {
	encoder := NewResp2Encoder()

	tests := []struct {
		name   string
		values [][]byte
		want   string
	}{
		{
			name: "SET command",
			values: [][]byte{
				[]byte("SET"),
				[]byte("user"),
				[]byte("1"),
			},
			want: "*3\r\n$3\r\nSET\r\n$4\r\nuser\r\n$1\r\n1\r\n",
		},
		{
			name:   "empty array",
			values: [][]byte{},
			want:   "*0\r\n",
		},
		{
			name: "empty string element",
			values: [][]byte{
				[]byte("ECHO"),
				[]byte(""),
			},
			want: "*2\r\n$4\r\nECHO\r\n$0\r\n\r\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := encoder.EncodeBulkStringArray(tt.values)

			if !bytes.Equal(got, []byte(tt.want)) {
				t.Fatalf("expected %q, got %q", tt.want, got)
			}
		})
	}
}
