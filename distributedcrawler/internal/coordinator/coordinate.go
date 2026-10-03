// Package coordinator: API → durable crawl request → orchestrator → URL jobs → workers → durable page results → orchestrator
package coordinator

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/goware/urlx"
	"github.com/nats-io/nats.go/jetstream"

	"github.com/julianschreiner/distributedcrawler/internal/messaging"
	"github.com/julianschreiner/distributedcrawler/internal/shared"
)

type Coordinator struct {
	Messenger *messaging.Messenger
}

func New(messenger *messaging.Messenger) (*Coordinator, error) {
	return &Coordinator{
		Messenger: messenger,
	}, nil
}

func (c *Coordinator) Run(ctx context.Context) error {
	requestConsumer, err := c.Messenger.JetStream.Consumer(
		ctx,
		messaging.CrawlRequestStream,
		messaging.CrawlRequestConsumer,
	)
	if err != nil {
		return fmt.Errorf("get crawl request consumer: %w", err)
	}

	requestMessage, err := requestConsumer.Consume(
		func(msg jetstream.Msg) {
			c.requestConsumeCallback(ctx, msg)
		},
		jetstream.ConsumeErrHandler(func(_ jetstream.ConsumeContext, err error) {
			log.Printf("crawl request consumer error: %v", err)
		}),
	)
	if err != nil {
		return fmt.Errorf("get crawl start consume: %w", err)
	}
	defer requestMessage.Stop()

	resultConsumer, err := c.Messenger.JetStream.Consumer(
		ctx,
		messaging.CrawlResultStream,
		messaging.CrawlResultConsumer,
	)
	if err != nil {
		return fmt.Errorf("get crawl result consumer: %w", err)
	}

	resultMessage, err := resultConsumer.Consume(
		func(msg jetstream.Msg) {
			c.resultConsumeCallback(ctx, msg)
		},
		jetstream.ConsumeErrHandler(func(_ jetstream.ConsumeContext, err error) {
			log.Printf("crawl result consumer error: %v", err)
		}),
	)
	if err != nil {
		return fmt.Errorf("get crawl result consume: %w", err)
	}
	defer resultMessage.Stop()
	<-ctx.Done()

	return nil
}

func (c *Coordinator) HandleStart(ctx context.Context, req shared.CrawlRequest, eventID uuid.UUID) error {
	err := c.setCrawlStatus(ctx, eventID, shared.StatusPending)
	if err != nil {
		return fmt.Errorf("set status crawl request: %w", err)
	}

	err = c.scheduleURL(ctx, eventID, req.StartURL, 0)
	if err != nil {
		return fmt.Errorf("schedule url: %w", err)
	}

	return nil
}

func (c *Coordinator) HandlePageResult(ctx context.Context, result shared.PageResult) error {
	panic("impement me")
}

func (c *Coordinator) scheduleURL(ctx context.Context, crawlID uuid.UUID, url string, depth int) error {
	urlParse, _ := urlx.Parse(url)
	normalizedURL, _ := urlx.Normalize(urlParse)
	sum := sha256.Sum256([]byte(crawlID.String() + "\x00" + normalizedURL))
	messageID := hex.EncodeToString(sum[:])

	urlJob := shared.URLJob{
		CrawlID: crawlID,
		URL:     normalizedURL,
		Depth:   depth,
	}

	message, err := json.Marshal(urlJob)
	if err != nil {
		return fmt.Errorf("marshal URL job: %w", err)
	}

	_, err = c.Messenger.JetStream.Publish(
		ctx,
		messaging.CrawlJobSubjectCreated,
		message,
		jetstream.WithMsgID(messageID),
	)
	if err != nil {
		return fmt.Errorf("publish URL job: %w", err)
	}

	return nil
}

func (c *Coordinator) tryComplete(ctx context.Context, crawlID uuid.UUID) error {
	panic("impement me")
}

func (c *Coordinator) requestConsumeCallback(ctx context.Context, msg jetstream.Msg) {
	var crawlStart shared.CrawlStartRequest
	if err := json.Unmarshal(msg.Data(), &crawlStart); err != nil {
		log.Printf("invalid crawl request %v", err)
		err = msg.Term() // should not be retried
		if err != nil {
			log.Printf("failed calling Term %v", err)
		}
		return
	}

	if err := c.HandleStart(ctx, crawlStart.Request, crawlStart.CrawlID); err != nil {
		log.Printf("start crawl: %v", err)
		err = msg.NakWithDelay(2 * time.Minute) // redeliver
		if err != nil {
			log.Printf("failed calling Nak %v", err)
		}
		return
	}

	err := msg.Ack()
	if err != nil {
		log.Printf("failed calling Ack %v", err)
	}
}

func (c *Coordinator) resultConsumeCallback(ctx context.Context, msg jetstream.Msg) {
	var crawlResult shared.PageResult
	if err := json.Unmarshal(msg.Data(), &crawlResult); err != nil {
		log.Printf("invalid crawl result %v", err)
		err = msg.Term() // should not be retried
		if err != nil {
			log.Printf("failed calling Term %v", err)
		}
		return
	}

	if err := c.HandlePageResult(ctx, crawlResult); err != nil {
		log.Printf("result crawl: %v", err)
		err = msg.NakWithDelay(2 * time.Minute) // redeliver
		if err != nil {
			log.Printf("failed calling Nak %v", err)
		}
		return
	}

	err := msg.Ack()
	if err != nil {
		log.Printf("failed calling Ack %v", err)
	}
}

func (c *Coordinator) setCrawlStatus(ctx context.Context, eventID uuid.UUID, jobStatus shared.JobStatus) error {
	crawlStatus := shared.CrawlStatus{
		CrawlID: eventID,
		Status:  jobStatus,
	}

	status, err := json.Marshal(crawlStatus)
	if err != nil {
		return fmt.Errorf("json marshal: %w", err)
	}

	key := eventID.String()
	if jobStatus == shared.StatusPending {
		_, err = c.Messenger.CrawlStatuses.Create(ctx, key, status)
		if errors.Is(err, jetstream.ErrKeyExists) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("create pending crawl status: %w", err)
		}
		return nil
	}

	entry, err := c.Messenger.CrawlStatuses.Get(ctx, key)
	if err != nil {
		return fmt.Errorf("get crawl status for update: %w", err)
	}
	if _, err := c.Messenger.CrawlStatuses.Update(ctx, key, status, entry.Revision()); err != nil {
		return fmt.Errorf("update crawl status: %w", err)
	}
	return nil
}
