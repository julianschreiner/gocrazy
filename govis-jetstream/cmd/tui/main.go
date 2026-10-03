package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/julianschreiner/govis-jetstream/internal/observe"
	"github.com/julianschreiner/govis-jetstream/internal/tui"
)

func main() {
	err := run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run() (retErr error) {
	natsConn := observe.NewClient()
	err := natsConn.ConnectNATS()
	if err != nil {
		return fmt.Errorf("connect nats: %w", err)
	}
	defer natsConn.Close()

	handler := &observe.EventHandler{
		Events: make(chan *observe.IncomingPublishEvent, 256),
	}
	ctx := context.Background()

	configuredSubjectPatters, err := natsConn.GetConfiguredSubjectPatternsForAllStreams(ctx)
	if err != nil {
		return fmt.Errorf("configured subject patterns: %w", err)
	}

	ctxSubscribe, subscribeCancel := context.WithTimeout(ctx, 5*time.Second)
	defer subscribeCancel()
	subs, err := handler.SubscribeForPublishEvents(
		ctxSubscribe,
		natsConn.NatsConn,
		configuredSubjectPatters,
	)
	if err != nil {
		return fmt.Errorf("subscribe for events: %w", err)
	}

	for _, sub := range subs {
		defer sub.Unsubscribe()
	}

	_, err = tea.NewProgram(
		tui.NewModel(natsConn, handler),
	).Run()
	if errors.Is(err, tea.ErrInterrupted) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("tea error: %w", err)
	}

	return nil
}
