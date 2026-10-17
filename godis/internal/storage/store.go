package storage

import (
	"bytes"
	"errors"
	"math"
	"regexp"
	"strconv"
	"time"
)

const DefaultTTL time.Duration = 1 * time.Hour

type Entry struct {
	Value     []byte
	ExpiresAt *time.Time
}

type Store struct {
	data map[string]Entry
}
type EntryResult struct {
	Data    *Entry
	Exists  bool
	Expired bool
}

func NewStore() *Store {
	return &Store{
		data: make(map[string]Entry),
	}
}

func (s *Store) Set(key string, value []byte, expiresAt *time.Time) {
	s.data[key] = Entry{
		Value:     bytes.Clone(value),
		ExpiresAt: expiresAt,
	}
}

func (s *Store) Get(key string) ([]byte, bool) {
	entry, exists := s.data[key]
	if !exists {
		return nil, false
	}

	// lazy expiration
	if entry.ExpiresAt != nil &&
		!time.Now().Before(*entry.ExpiresAt) {
		s.Delete(key)
		return nil, false
	}

	return bytes.Clone(entry.Value), true
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
	entry, exists := s.data[key]
	if entry.ExpiresAt != nil && !time.Now().Before(*entry.ExpiresAt) {
		delete(s.data, key)
		return false
	}
	return exists
}

func (s *Store) Incr(key string) (int64, error) {
	var number int64

	entry, exists := s.data[key]
	if exists {
		if entry.ExpiresAt != nil && !time.Now().Before(*entry.ExpiresAt) {
			delete(s.data, key)
			return 0, errors.New("entry is already expired")
		}

		parsed, err := strconv.ParseInt(string(entry.Value), 10, 64)
		if err != nil {
			return 0, errors.New("value is not an integer or out of range")
		}
		number = parsed
	}

	if number == math.MaxInt64 {
		return 0, errors.New("increment would overflow int64")
	}

	number++

	entry.Value = []byte(strconv.FormatInt(number, 10))
	s.data[key] = entry

	return number, nil
}

func (s *Store) Expire(key string, duration time.Time) error {
	panic("implement me")
}

func (s *Store) CleanupExpired() {
	for key, entry := range s.data {
		if entry.ExpiresAt != nil && !time.Now().Before(*entry.ExpiresAt) {
			delete(s.data, key)
		}
	}
}

func (s *Store) GetKeys(pattern string) ([][]byte, bool, error) {
	var result [][]byte
	regex, err := regexp.Compile(pattern)
	if err != nil {
		return nil, false, err
	}

	for key := range s.data {
		transformedKey := []byte(key)
		if regex.Match(transformedKey) {
			result = append(result, transformedKey)
		}
	}

	if len(result) == 0 {
		return nil, false, err
	}

	return result, true, nil
}
