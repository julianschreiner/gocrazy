package storage

import (
	"bytes"
	"errors"
	"math"
	"strconv"
)

type Store struct {
	data map[string][]byte
}

func NewStore() *Store {
	return &Store{
		data: make(map[string][]byte),
	}
}

func (s *Store) Set(key string, value []byte) {
	s.data[key] = bytes.Clone(value)
}

func (s *Store) Get(key string) ([]byte, bool) {
	val, exists := s.data[key]
	if !exists {
		return nil, false
	}

	return bytes.Clone(val), true
}

func (s *Store) Delete(key string) bool {
	_, exists := s.data[key]
	if !exists {
		return false
	}

	delete(s.data, key)
	return true
}

func (s *Store) Exists(key string) bool {
	_, exists := s.data[key]
	return exists
}

func (s *Store) Incr(key string) (int64, error) {
	var number int64

	val, exists := s.data[key]
	if exists {
		parsed, err := strconv.ParseInt(string(val), 10, 64)
		if err != nil {
			return 0, errors.New("value is not an integer or out of range")
		}
		number = parsed
	}

	if number == math.MaxInt64 {
		return 0, errors.New("increment would overflow int64")
	}

	number++

	s.Set(key, []byte(strconv.FormatInt(number, 10)))

	return number, nil
}
