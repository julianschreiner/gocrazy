package observe

// nats connection and read-only jetstream queries
//
// Use the repo’s existing `github.com/nats-io/nats.go/jetstream`
// dependency for read-only state queries: `js.ListStreams(ctx)`,
// `stream.Info(ctx)`, `stream.ConsumerNames(ctx)`, and
// `consumer.Info(ctx)`. Use ordinary `nc.Subscribe(...)` for
// `observe.>` and the `$JS.EVENT...` subjects.
// This keeps the TUI out of the work queue.
// [Go JetStream API]
// (<https://github.com/nats-io/nats.go/blob/main/jetstream/README.md>)
//
//
//

import (
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

type NatsClient struct {
	NatsConn  *nats.Conn
	JetStream jetstream.JetStream
}

func NewClient() *NatsClient {
	return &NatsClient{}
}

func (n *NatsClient) ConnectNATS() error {
	if n.NatsConn != nil {
		return nil
	}

	nc, err := nats.Connect("nats://127.0.0.1:4222", nats.Name("govis-tui"))
	if err != nil {
		return err
	}

	js, err := jetstream.New(nc)
	if err != nil {
		nc.Close()
		return err
	}

	n.NatsConn = nc
	n.JetStream = js
	return nil
}

func (n *NatsClient) Close() {
	if n.NatsConn != nil {
		n.NatsConn.Close()
		n.NatsConn = nil
		n.JetStream = nil
	}
}

func (n *NatsClient) ListStreams(ctx context.Context) ([]StreamSummary, error) {
	lister := n.JetStream.ListStreams(ctx)

	var streams []StreamSummary
	for info := range lister.Info() {
		streams = append(streams, StreamSummary{
			Name:         info.Config.Name,
			CreatedAt:    info.Created,
			NumConsumers: info.State.Consumers,
			NumMessages:  info.State.Msgs,
			FirstSeq:     info.State.FirstSeq,
			LastSeq:      info.State.LastSeq,
			MaxConsumers: info.Config.MaxConsumers,
			MaxMessages:  info.Config.MaxMsgs,
		})
	}

	if err := lister.Err(); err != nil {
		return nil, fmt.Errorf("list streams: %w", err)
	}

	return streams, nil
}

func (n *NatsClient) StreamSubjects(ctx context.Context, name string) ([]string, error) {
	stream, err := n.JetStream.Stream(ctx, name)
	if err != nil {
		return nil, fmt.Errorf("stream %s: %w", name, err)
	}

	info, err := stream.Info(ctx, jetstream.WithSubjectFilter(">"))
	if err != nil {
		return nil, fmt.Errorf("stream %s subjects: %w", name, err)
	}
	var subjects []string
	for subject := range info.State.Subjects {
		subjects = append(subjects, subject)
	}
	slices.Sort(subjects)
	return subjects, nil
}

func (n *NatsClient) ListConsumers(ctx context.Context, name string) ([]string, error) {
	stream, err := n.JetStream.Stream(ctx, name)
	if err != nil {
		return nil, fmt.Errorf("stream %s: %w", name, err)
	}

	var consumerSummaries []string
	consumerLister := stream.ConsumerNames(ctx)

	for consumerInfo := range consumerLister.Name() {
		consumerSummaries = append(consumerSummaries, consumerInfo)
	}

	slices.Sort(consumerSummaries)

	return consumerSummaries, nil
}

func (n *NatsClient) GetConsumerSnapshot(ctx context.Context, streamName, consumerName string) (*ConsumerSnapshot, error) {
	consumer, err := n.JetStream.Consumer(ctx, streamName, consumerName)
	if err != nil {
		return nil, fmt.Errorf("consumer snapshot %s: %w", consumerName, err)
	}

	consumerInfo, err := consumer.Info(ctx)
	if err != nil {
		return nil, fmt.Errorf("consumer info %s: %w", consumerName, err)
	}

	return &ConsumerSnapshot{
		Stream:               streamName,
		Consumer:             consumerName,
		ObservedAt:           time.Now(),
		DeliveredConsumerSeq: consumerInfo.Delivered.Consumer,
		DeliveredStreamSeq:   consumerInfo.Delivered.Stream,
		AckFloorConsumerSeq:  consumerInfo.AckFloor.Consumer,
		AckFloorStreamSeq:    consumerInfo.AckFloor.Stream,
		NumPending:           consumerInfo.NumPending,
		NumAckPending:        consumerInfo.NumAckPending,
	}, nil
}

func (n *NatsClient) GetConfiguredSubjectPatternsForAllStreams(ctx context.Context) ([]ConfiguredStreamSubjectPatterns, error) {
	lister := n.JetStream.ListStreams(ctx)

	var configuredSubjectPatterns []ConfiguredStreamSubjectPatterns
	for stream := range lister.Info() {
		configuredSubjectPatterns = append(configuredSubjectPatterns, ConfiguredStreamSubjectPatterns{
			Stream:         stream.Config.Name,
			SubjectPattern: stream.Config.Subjects,
		})
	}

	if err := lister.Err(); err != nil {
		return nil, fmt.Errorf("list streams: %w", err)
	}

	return configuredSubjectPatterns, nil
}
