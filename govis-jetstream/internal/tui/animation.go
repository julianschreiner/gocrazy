package tui

import (
	"math"
	"strings"
	"time"

	"github.com/julianschreiner/govis-jetstream/internal/observe"
)

const (
	flowLaneWidth        = 24
	maxParticlesPerLane  = 24
	maxDeliveryParticles = 12
)

type flowParticle struct {
	startedAt time.Time
	duration  time.Duration
}

func (m *Model) resetFlow() {
	m.animationNow = time.Time{}
	m.publishParticles = nil
	m.consumerParticles = nil
	m.consumerRates = nil
	m.publishWindowStart = time.Time{}
	m.publishWindowCount = 0
	m.publishRate = 0
	m.newEventStream = ""
	m.newEventSubject = ""
	m.newEventJobID = ""
}

func (m *Model) advanceAnimation(now time.Time) {
	m.animationNow = now
	m.rollPublishWindow(now)
	m.publishParticles = activeParticles(m.publishParticles, now)
	for consumer, particles := range m.consumerParticles {
		active := activeParticles(particles, now)
		if len(active) == 0 {
			delete(m.consumerParticles, consumer)
		} else {
			m.consumerParticles[consumer] = active
		}
	}
}

func (m *Model) rollPublishWindow(now time.Time) {
	if m.publishWindowStart.IsZero() {
		m.publishWindowStart = now
		return
	}
	if elapsed := now.Sub(m.publishWindowStart); elapsed >= time.Second {
		m.publishRate = float64(m.publishWindowCount) / elapsed.Seconds()
		m.publishWindowStart = now
		m.publishWindowCount = 0
	}
}

func (m *Model) currentPublishRate() float64 {
	if m.publishWindowStart.IsZero() || m.publishWindowCount == 0 {
		return m.publishRate
	}
	elapsed := m.animationNow.Sub(m.publishWindowStart).Seconds()
	return math.Max(m.publishRate, float64(m.publishWindowCount)/math.Max(elapsed, 0.25))
}

func (m *Model) recordPublish(now time.Time) {
	m.rollPublishWindow(now)
	m.animationNow = now
	m.publishWindowCount++
	m.publishParticles = appendLimited(m.publishParticles, flowParticle{
		startedAt: now,
		duration:  flowDuration(m.currentPublishRate()),
	})
}

func (m *Model) recordConsumerSnapshot(previous, current *observe.ConsumerSnapshot) {
	if current == nil {
		return
	}
	if m.consumerRates == nil {
		m.consumerRates = make(map[string]float64)
	}
	m.consumerRates[current.Consumer] = 0
	if previous == nil || current.DeliveredConsumerSeq <= previous.DeliveredConsumerSeq {
		return
	}

	elapsed := current.ObservedAt.Sub(previous.ObservedAt).Seconds()
	if elapsed <= 0 {
		return
	}
	deliveries := current.DeliveredConsumerSeq - previous.DeliveredConsumerSeq
	rate := float64(deliveries) / elapsed
	m.consumerRates[current.Consumer] = rate
	if m.consumerParticles == nil {
		m.consumerParticles = make(map[string][]flowParticle)
	}

	count := int(min(deliveries, maxDeliveryParticles))
	now := time.Now()
	for i := range count {
		start := now
		if count > 1 {
			start = start.Add(time.Duration(i) * 800 * time.Millisecond / time.Duration(count-1))
		}
		m.consumerParticles[current.Consumer] = appendLimited(m.consumerParticles[current.Consumer], flowParticle{
			startedAt: start,
			duration:  flowDuration(rate),
		})
	}
}

func flowDuration(rate float64) time.Duration {
	duration := time.Duration(float64(1400*time.Millisecond) / (1 + rate/8))
	if duration < 350*time.Millisecond {
		return 350 * time.Millisecond
	}
	return duration
}

func appendLimited(particles []flowParticle, particle flowParticle) []flowParticle {
	particles = append(particles, particle)
	if len(particles) > maxParticlesPerLane {
		return particles[len(particles)-maxParticlesPerLane:]
	}
	return particles
}

func activeParticles(particles []flowParticle, now time.Time) []flowParticle {
	active := particles[:0]
	for _, particle := range particles {
		if now.Before(particle.startedAt.Add(particle.duration)) {
			active = append(active, particle)
		}
	}
	return active
}

func flowLane(particles []flowParticle, now time.Time) string {
	track := []byte(strings.Repeat("-", flowLaneWidth))
	for _, particle := range particles {
		elapsed := now.Sub(particle.startedAt)
		if elapsed < 0 || elapsed >= particle.duration || particle.duration <= 0 {
			continue
		}
		position := int(float64(elapsed) / float64(particle.duration) * float64(flowLaneWidth-1))
		track[position] = 'o'
	}
	return "[" + string(track) + ">]"
}
