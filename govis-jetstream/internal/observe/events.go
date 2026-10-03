package observe

import (
	"context"

	"github.com/nats-io/nats.go"
)

type IncomingPublishEvent struct {
	Stream  string
	Subject string
	JobID   string
}

type EventHandler struct {
	Events chan *IncomingPublishEvent
}

func (e *EventHandler) SubscribeForPublishEvents(ctx context.Context, nc *nats.Conn, configuredSubjects []ConfiguredStreamSubjectPatterns) ([]*nats.Subscription, error) {
	var subscriptions []*nats.Subscription
	for _, config := range configuredSubjects {
		for _, subject := range config.SubjectPattern {
			sub, err := nc.Subscribe(subject, func(msg *nats.Msg) {
				e.publishEventCallBack(config.Stream, msg)
			})
			if err != nil {
				for _, existing := range subscriptions {
					_ = existing.Unsubscribe()
				}
				return nil, err
			}
			subscriptions = append(subscriptions, sub)
		}
	}

	err := nc.FlushWithContext(ctx)
	if err != nil {
		for _, existing := range subscriptions {
			_ = existing.Unsubscribe()
		}
		return nil, err
	}

	return subscriptions, nil
}

func (e *EventHandler) publishEventCallBack(streamName string, msg *nats.Msg) {
	incomingEvent := &IncomingPublishEvent{
		Stream:  streamName,
		Subject: msg.Subject,
		JobID:   msg.Header.Get("Job-Id"),
	}

	select {
	case e.Events <- incomingEvent:
	default:
		// Channel full; count dropped, display events if needed
	}
}

func (e *EventHandler) SubscribeForAckMetrics(ctx context.Context, nc *nats.Conn) ([]*nats.Subscription, error) {
	// TODO ack events where consumer sampling is configured
	// sub, err := nc.Subscribe("$JS.EVENT.METRIC.>", e.ackMetricCallback
	panic("not implemented yet")
}
