package tui

import (
	"context"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/julianschreiner/govis-jetstream/internal/observe"
)

func refreshTick() tea.Cmd {
	return tea.Tick(time.Second, func(time.Time) tea.Msg {
		return refreshTickMsg{}
	})
}

func animationTick() tea.Cmd {
	return tea.Tick(100*time.Millisecond, func(t time.Time) tea.Msg {
		return animationTickMsg(t)
	})
}

func NewModel(client *observe.NatsClient, handler *observe.EventHandler) Model {
	return Model{
		client:          client,
		eventHandler:    handler,
		streamsLoading:  true,
		streamsFetching: true,
	}
}

func (m Model) loadStreams() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		streams, err := m.client.ListStreams(ctx)
		return streamsLoadedMsg{streams: streams, err: err}
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(m.loadStreams(), refreshTick(), animationTick(), m.WaitForPublishEvents())
}

func (m Model) loadSubjects(stream string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		subjects, err := m.client.StreamSubjects(ctx, stream)
		return subjectsLoadedMsg{stream: stream, subjects: subjects, err: err}
	}
}

func (m Model) loadConsumers(stream string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		consumers, err := m.client.ListConsumers(ctx, stream)
		return consumersLoadedMsg{stream: stream, consumers: consumers, err: err}
	}
}

func (m Model) selectedStream() string {
	if m.selected < 0 || m.selected >= len(m.streams) {
		return ""
	}
	return m.streams[m.selected].Name
}

func (m Model) WaitForPublishEvents() tea.Cmd {
	return func() tea.Msg {
		event := <-m.eventHandler.Events
		return publishEventReceived{incomingEvent: event}
	}
}

func (m Model) loadConsumerSnapshots(stream string, consumerNames []string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		consumerSnapshots := make([]*observe.ConsumerSnapshot, 0, len(consumerNames))
		for _, consumerName := range consumerNames {
			consumerSnapshot, err := m.client.GetConsumerSnapshot(ctx, stream, consumerName)
			if err != nil {
				return consumerSnapshotsLoadedMsg{stream: stream, err: err}
			}
			consumerSnapshots = append(consumerSnapshots, consumerSnapshot)
		}

		return consumerSnapshotsLoadedMsg{stream: stream, consumerSnapshots: consumerSnapshots}
	}
}
