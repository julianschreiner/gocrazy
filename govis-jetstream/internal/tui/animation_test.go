package tui

import (
	"testing"
	"time"

	"github.com/julianschreiner/govis-jetstream/internal/observe"
)

func TestConsumerAnimationStartsFromNewDeliveries(t *testing.T) {
	now := time.Now()
	m := Model{}
	first := &observe.ConsumerSnapshot{Consumer: "EMAIL", ObservedAt: now, DeliveredConsumerSeq: 100}
	m.recordConsumerSnapshot(nil, first)
	if len(m.consumerParticles["EMAIL"]) != 0 {
		t.Fatal("initial snapshot animated earlier deliveries")
	}

	second := &observe.ConsumerSnapshot{Consumer: "EMAIL", ObservedAt: now.Add(time.Second), DeliveredConsumerSeq: 103}
	m.recordConsumerSnapshot(first, second)
	if got := len(m.consumerParticles["EMAIL"]); got != 3 {
		t.Fatalf("expected 3 delivery particles, got %d", got)
	}
	if got := m.consumerRates["EMAIL"]; got != 3 {
		t.Fatalf("expected 3 deliveries per second, got %v", got)
	}
}
