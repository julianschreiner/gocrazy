// Package messaging sets up connection to nats jetstream and registers streams/workers
package messaging

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

const (
	CrawlRequestStream      = "CRAWL_REQUEST"
	CrawlRequestConsumer    = "CRAWL_REQUEST_CONSUMER"
	CrawlRequestSubject     = "crawl.request.created"
	CrawlJobStream          = "CRAWL_JOB"
	CrawlJobConsumer        = "CRAWL_JOB_CONSUMER"
	CrawlJobSubjectWildcard = "crawl.job.>"
	CrawlJobSubjectCreated  = "crawl.job.created"
	CrawlResultStream       = "CRAWL_RESULT"
	CrawlResultConsumer     = "CRAWL_RESULT_CONSUMER"
	CrawlResultSubject      = "crawl.result.created"
	CrawlStatusBucket       = "CRAWL_STATUS"
)

type Messenger struct {
	NATSConnection *nats.Conn
	JetStream      jetstream.JetStream
	CrawlStatuses  jetstream.KeyValue
}

func New() (*Messenger, error) {
	nc, err := nats.Connect(
		"localhost:4222",
		nats.Name("distributed-crawler-api"),
		nats.Timeout(5*time.Second),
	)
	if err != nil {
		return nil, err
	}

	js, err := jetstream.New(nc, jetstream.WithPublishAsyncMaxPending(256))
	if err != nil {
		nc.Close()
		return nil, err
	}

	return &Messenger{
		NATSConnection: nc,
		JetStream:      js,
	}, nil
}

func (m *Messenger) EnsureInfrastructure(ctx context.Context) error {
	err := m.ensureCrawlRequest(ctx)
	if err != nil {
		return err
	}

	err = m.ensureCrawlJob(ctx)
	if err != nil {
		return err
	}

	err = m.ensureCrawlResult(ctx)
	if err != nil {
		return err
	}

	err = m.ensureStatusBucket(ctx)
	if err != nil {
		return err
	}

	return nil
}

func (m *Messenger) ensureCrawlRequest(ctx context.Context) error {
	_, err := m.JetStream.Stream(ctx, CrawlRequestStream)
	if errors.Is(err, jetstream.ErrStreamNotFound) {
		_, err = m.JetStream.CreateStream(ctx, jetstream.StreamConfig{
			Name:      CrawlRequestStream,
			Subjects:  []string{CrawlRequestSubject},
			Retention: jetstream.WorkQueuePolicy,
			Storage:   jetstream.FileStorage,
			MaxAge:    30 * 24 * time.Hour,
		})
	}
	if err != nil {
		return fmt.Errorf("ensure crawl request stream: %w", err)
	}

	_, err = m.JetStream.Consumer(ctx, CrawlRequestStream, CrawlRequestConsumer)
	if errors.Is(err, jetstream.ErrConsumerNotFound) {
		_, err = m.JetStream.CreateConsumer(ctx, CrawlRequestStream, jetstream.ConsumerConfig{
			Name:          CrawlRequestConsumer,
			Durable:       CrawlRequestConsumer,
			AckPolicy:     jetstream.AckExplicitPolicy,
			FilterSubject: CrawlRequestSubject,
		})
	}
	if err != nil {
		return fmt.Errorf("adding consumer %s to stream %s failed: %w", CrawlRequestConsumer, CrawlRequestStream, err)
	}

	return nil
}

func (m *Messenger) ensureCrawlJob(ctx context.Context) error {
	_, err := m.JetStream.Stream(ctx, CrawlJobStream)
	if errors.Is(err, jetstream.ErrStreamNotFound) {
		_, err = m.JetStream.CreateStream(
			ctx,
			jetstream.StreamConfig{
				Name:      CrawlJobStream,
				Retention: jetstream.WorkQueuePolicy,
				Subjects:  []string{CrawlJobSubjectWildcard},
				Storage:   jetstream.FileStorage,
				MaxAge:    30 * 24 * time.Hour,
			},
		)
	}
	if err != nil {
		return fmt.Errorf("ensure crawl job stream: %w", err)
	}

	_, err = m.JetStream.Consumer(ctx, CrawlJobStream, CrawlJobConsumer)
	if errors.Is(err, jetstream.ErrConsumerNotFound) {
		_, err = m.JetStream.CreateConsumer(ctx, CrawlJobStream, jetstream.ConsumerConfig{
			Name:          CrawlJobConsumer,
			Durable:       CrawlJobConsumer,
			AckPolicy:     jetstream.AckExplicitPolicy,
			FilterSubject: CrawlJobSubjectWildcard,
		})
	}
	if err != nil {
		return fmt.Errorf("adding consumer %s to stream %s failed: %w", CrawlJobConsumer, CrawlJobStream, err)
	}

	return nil
}

func (m *Messenger) ensureCrawlResult(ctx context.Context) error {
	_, err := m.JetStream.Stream(ctx, CrawlResultStream)
	if errors.Is(err, jetstream.ErrStreamNotFound) {
		_, err = m.JetStream.CreateStream(
			ctx,
			jetstream.StreamConfig{
				Name:      CrawlResultStream,
				Retention: jetstream.WorkQueuePolicy,
				Subjects:  []string{CrawlResultSubject},
				Storage:   jetstream.FileStorage,
				MaxAge:    30 * 24 * time.Hour,
			},
		)
	}
	if err != nil {
		return fmt.Errorf("ensure result job stream: %w", err)
	}

	_, err = m.JetStream.Consumer(ctx, CrawlResultStream, CrawlResultConsumer)
	if errors.Is(err, jetstream.ErrConsumerNotFound) {
		_, err = m.JetStream.CreateConsumer(ctx, CrawlResultStream, jetstream.ConsumerConfig{
			Name:          CrawlResultConsumer,
			Durable:       CrawlResultConsumer,
			AckPolicy:     jetstream.AckExplicitPolicy,
			FilterSubject: CrawlResultSubject,
		})
	}
	if err != nil {
		return fmt.Errorf("adding consumer %s to stream %s failed: %w", CrawlResultConsumer, CrawlResultStream, err)
	}

	return nil
}

// EnsureAPIStreamAndBucket this method verifies API dependencies
func (m *Messenger) EnsureAPIStreamAndBucket(ctx context.Context) error {
	_, err := m.JetStream.Stream(ctx, CrawlRequestStream)
	if errors.Is(err, jetstream.ErrStreamNotFound) {
		return err
	}
	if err != nil {
		return err
	}

	kv, err := m.JetStream.KeyValue(ctx, CrawlStatusBucket)
	if errors.Is(err, jetstream.ErrBucketNotFound) {
		return err
	}
	if err != nil {
		return err
	}
	// each process handles its own kv handle, this one is for api
	m.CrawlStatuses = kv

	return nil
}

func (m *Messenger) ensureStatusBucket(ctx context.Context) error {
	kv, err := m.JetStream.KeyValue(ctx, CrawlStatusBucket)

	if errors.Is(err, jetstream.ErrBucketNotFound) {
		kv, err = m.JetStream.CreateKeyValue(ctx, jetstream.KeyValueConfig{
			Bucket:  CrawlStatusBucket,
			History: 1,
			TTL:     30 * 24 * time.Hour,
			Storage: jetstream.FileStorage,
		})
	}

	if err != nil {
		return fmt.Errorf("ensure crawl status bucket: %w", err)
	}

	// each process handles its own kv handle, this one is for orchestrator
	m.CrawlStatuses = kv

	return nil
}
