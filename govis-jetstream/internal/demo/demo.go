package demo

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/nats-io/nats.go/jetstream"
)

const (
	Stream       = "JOBS"
	EventsStream = "EVENTS"
)

var Kinds = []string{"email", "report", "image"}

type Job struct {
	ID      string `json:"id"`
	Kind    string `json:"kind"`
	Outcome string `json:"outcome"`
}

func Outcome(seq uint64) string {
	switch seq % 20 {
	case 0:
		return "term"
	case 1, 2, 3:
		return "nak-once"
	default:
		return "ack"
	}
}

func NewJob(runID string, seq uint64) Job {
	return Job{
		ID:      fmt.Sprintf("%s-%d", runID, seq),
		Kind:    Kinds[(seq-1)%uint64(len(Kinds))],
		Outcome: Outcome(seq),
	}
}

func Decode(data []byte) (Job, error) {
	var job Job
	if err := json.Unmarshal(data, &job); err != nil {
		return Job{}, err
	}
	if job.ID == "" || job.Kind == "" {
		return Job{}, fmt.Errorf("missing job identity")
	}
	return job, nil
}

func ConsumerName(kind string) (string, error) {
	switch kind {
	case "email":
		return "EMAIL", nil
	case "report":
		return "REPORT", nil
	case "image":
		return "IMAGE", nil
	default:
		return "", fmt.Errorf("unknown worker kind %q", kind)
	}
}

func Setup(ctx context.Context, js jetstream.JetStream) error {
	stream, err := js.CreateOrUpdateStream(ctx, jetstream.StreamConfig{
		Name:      Stream,
		Subjects:  []string{"jobs.*"},
		Retention: jetstream.WorkQueuePolicy,
		Storage:   jetstream.FileStorage,
		MaxAge:    time.Hour,
		MaxMsgs:   20000,
		RePublish: &jetstream.RePublish{
			Source: "jobs.>", Destination: "observe.>", HeadersOnly: true,
		},
	})
	if err != nil {
		return fmt.Errorf("create stream: %w", err)
	}
	for _, kind := range Kinds {
		name, _ := ConsumerName(kind)
		_, err := stream.CreateOrUpdateConsumer(ctx, jetstream.ConsumerConfig{
			Durable:         name,
			FilterSubject:   "jobs." + kind,
			AckPolicy:       jetstream.AckExplicitPolicy,
			AckWait:         10 * time.Second,
			MaxDeliver:      3,
			MaxAckPending:   100,
			SampleFrequency: "100%",
		})
		if err != nil {
			return fmt.Errorf("create %s consumer: %w", name, err)
		}
	}

	events, err := js.CreateOrUpdateStream(ctx, jetstream.StreamConfig{
		Name:         EventsStream,
		Subjects:     []string{"events.*"},
		Retention:    jetstream.LimitsPolicy,
		Storage:      jetstream.FileStorage,
		MaxConsumers: 3,
		MaxMsgs:      1000,
		MaxAge:       time.Hour,
	})
	if err != nil {
		return fmt.Errorf("create %s stream: %w", EventsStream, err)
	}
	for _, consumer := range []struct {
		name   string
		filter string
	}{
		{name: "ALL", filter: "events.*"},
		{name: "EMAIL", filter: "events.email"},
		{name: "REPORT", filter: "events.report"},
	} {
		_, err := events.CreateOrUpdateConsumer(ctx, jetstream.ConsumerConfig{
			Durable:       consumer.name,
			FilterSubject: consumer.filter,
			AckPolicy:     jetstream.AckExplicitPolicy,
		})
		if err != nil {
			return fmt.Errorf("create %s consumer on %s: %w", consumer.name, EventsStream, err)
		}
	}
	return nil
}
