package storage

import (
	"bytes"
	"math"
	"strconv"
	"testing"
)

func TestNewStore(t *testing.T) {
	store := NewStore()

	if store == nil {
		t.Fatal("expected store to be initialized")
	}

	if store.data == nil {
		t.Fatal("expected data map to be initialized")
	}

	if len(store.data) != 0 {
		t.Fatalf("expected empty store, got %d entries", len(store.data))
	}
}

func TestSetAndGet(t *testing.T) {
	store := NewStore()

	store.Set("user", []byte("John"), nil)

	value, exists := store.Get("user")
	if !exists {
		t.Fatal("expected key 'user' to exist")
	}

	if !bytes.Equal(value, []byte("John")) {
		t.Fatalf("expected John, got %q", value)
	}
}

func TestSetOverwritesExistingValue(t *testing.T) {
	store := NewStore()

	store.Set("user", []byte("John"), nil)
	store.Set("user", []byte("Jane"), nil)

	value, exists := store.Get("user")
	if !exists {
		t.Fatal("expected key 'user' to exist")
	}

	if !bytes.Equal(value, []byte("Jane")) {
		t.Fatalf("expected Jane, got %q", value)
	}

	if len(store.data) != 1 {
		t.Fatalf("expected 1 key, got %d", len(store.data))
	}
}

func TestGetNonexistentKey(t *testing.T) {
	store := NewStore()

	value, exists := store.Get("missing")

	if exists {
		t.Fatal("expected missing key not to exist")
	}

	if value != nil {
		t.Fatalf("expected nil value, got %q", value)
	}
}

func TestDelete(t *testing.T) {
	store := NewStore()
	store.Set("user", []byte("John"), nil)

	if !store.Delete("user") {
		t.Fatal("expected deletion of existing key to return true")
	}

	if store.Exists("user") {
		t.Fatal("expected key to be removed")
	}

	if store.Delete("user") {
		t.Fatal("expected second deletion to return false")
	}
}

func TestExists(t *testing.T) {
	store := NewStore()

	if store.Exists("user") {
		t.Fatal("expected key not to exist")
	}

	store.Set("user", []byte("John"), nil)

	if !store.Exists("user") {
		t.Fatal("expected key to exist")
	}

	store.Delete("user")

	if store.Exists("user") {
		t.Fatal("expected key not to exist after deletion")
	}
}

func TestEmptyAndBinarySafeValues(t *testing.T) {
	tests := []struct {
		name  string
		value []byte
	}{
		{"empty value", []byte{}},
		{"null bytes", []byte{0, 1, 0, 2}},
		{"embedded CRLF", []byte("hello\r\nworld")},
		{"binary data", []byte{0x00, 0xFF, 0x80, 0x01}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := NewStore()
			store.Set("key", tt.value, nil)

			got, exists := store.Get("key")
			if !exists {
				t.Fatal("expected key to exist")
			}

			if !bytes.Equal(got, tt.value) {
				t.Fatalf("expected %v, got %v", tt.value, got)
			}
		})
	}
}

func TestSetCopiesInput(t *testing.T) {
	store := NewStore()
	original := []byte("John")

	store.Set("user", original, nil)

	// Modify the original input after storing it.
	original[0] = 'X'

	value, _ := store.Get("user")
	if !bytes.Equal(value, []byte("John")) {
		t.Fatalf("stored value was modified: got %q", value)
	}
}

func TestGetReturnsCopy(t *testing.T) {
	store := NewStore()
	store.Set("user", []byte("John"), nil)

	value, _ := store.Get("user")

	// Modify the returned slice.
	value[0] = 'X'

	// Stored data should remain unchanged.
	again, _ := store.Get("user")

	if !bytes.Equal(again, []byte("John")) {
		t.Fatalf("stored value was modified: got %q", again)
	}
}

func TestIncr(t *testing.T) {
	tests := []struct {
		name     string
		initial  []byte
		exists   bool
		expected int64
	}{
		{"missing key", nil, false, 1},
		{"zero", []byte("0"), true, 1},
		{"positive", []byte("41"), true, 42},
		{"negative", []byte("-5"), true, -4},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := NewStore()

			if tt.exists {
				store.Set("counter", tt.initial, nil)
			}

			result, err := store.Incr("counter")
			if err != nil {
				t.Fatal(err)
			}

			if result != tt.expected {
				t.Fatalf("expected %d, got %d", tt.expected, result)
			}

			// Verify the result was persisted in the map.
			value, exists := store.Get("counter")
			if !exists {
				t.Fatal("expected counter to exist after increment")
			}

			want := []byte(strconv.FormatInt(tt.expected, 10))
			if !bytes.Equal(value, want) {
				t.Fatalf("expected stored value %q, got %q", want, value)
			}
		})
	}
}

func TestIncrMultipleTimes(t *testing.T) {
	store := NewStore()

	for i := int64(1); i <= 10; i++ {
		result, err := store.Incr("counter")
		if err != nil {
			t.Fatal(err)
		}

		if result != i {
			t.Fatalf("expected %d, got %d", i, result)
		}
	}
}

func TestIncrInvalidValue(t *testing.T) {
	tests := []string{
		"hello",
		"12.5",
		"",
		"9223372036854775808",
		"1 2",
	}

	for _, value := range tests {
		t.Run(value, func(t *testing.T) {
			store := NewStore()
			store.Set("counter", []byte(value), nil)

			_, err := store.Incr("counter")
			if err == nil {
				t.Fatalf("expected error for value %q", value)
			}

			// Failed increment must not change stored value.
			stored, _ := store.Get("counter")
			if !bytes.Equal(stored, []byte(value)) {
				t.Fatalf("value changed after failed increment: %q", stored)
			}
		})
	}
}

func TestIncrOverflow(t *testing.T) {
	store := NewStore()
	store.Set("counter", []byte(strconv.FormatInt(math.MaxInt64, 10)), nil)

	_, err := store.Incr("counter")
	if err == nil {
		t.Fatal("expected overflow error")
	}

	// Original value must remain unchanged.
	value, _ := store.Get("counter")
	want := []byte(strconv.FormatInt(math.MaxInt64, 10))

	if !bytes.Equal(value, want) {
		t.Fatalf("expected %q, got %q", want, value)
	}
}
