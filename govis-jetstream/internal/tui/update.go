package tui

import (
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/julianschreiner/govis-jetstream/internal/observe"
)

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case animationTickMsg:
		m.advanceAnimation(time.Time(msg))
		return m, animationTick()

	case refreshTickMsg:
		// Tick fires once, so arm the next one. Poll each resource only when its previous request has finished.
		cmds := []tea.Cmd{refreshTick()}
		if !m.streamsFetching {
			m.streamsFetching = true
			cmds = append(cmds, m.loadStreams())
		}
		if !m.consumerPollFetching && m.selectedStream() != "" && len(m.consumers) > 0 {
			m.consumerPollFetching = true
			cmds = append(cmds, m.loadConsumerSnapshots(m.selectedStream(), m.consumers))
		}
		return m, tea.Batch(cmds...)
	case streamsLoadedMsg:
		m.streamsLoading = false
		m.streamsFetching = false
		m.streamsErr = msg.err
		if msg.err != nil {
			return m, nil
		}
		previous := m.selectedStream()
		m.streams = msg.streams
		m.selected = 0
		for i, stream := range m.streams {
			if stream.Name == previous {
				m.selected = i
				break
			}
		}
		if m.selectedStream() != previous {
			m.resetFlow()
			m.consumerSnapshots = nil
			m.consumerSnapshotsErr = nil
			m.subjects = nil
			m.subjectsErr = nil
			m.subjectsLoading = false
			m.consumers = nil
			m.consumersErr = nil
			m.consumersLoading = false
		}
		if stream := m.selectedStream(); stream != "" {
			var cmds []tea.Cmd
			if !m.subjectsLoading {
				m.subjectsLoading = true
				cmds = append(cmds, m.loadSubjects(stream))
			}
			if !m.consumersLoading {
				m.consumersLoading = true
				cmds = append(cmds, m.loadConsumers(stream))
			}
			return m, tea.Batch(cmds...)
		}

	case subjectsLoadedMsg:
		if msg.stream != m.selectedStream() {
			return m, nil // an earlier selection finished loading later
		}
		m.subjectsLoading = false
		m.subjects = msg.subjects
		m.subjectsErr = msg.err

	case consumersLoadedMsg:
		if msg.stream != m.selectedStream() {
			return m, nil // an earlier selection finished loading later
		}
		m.consumersLoading = false
		m.consumers = msg.consumers
		m.consumersErr = msg.err
		if len(msg.consumers) == 0 {
			m.consumerSnapshots = nil
			m.consumerParticles = nil
			m.consumerRates = nil
		}

	case publishEventReceived:
		if msg.incomingEvent == nil {
			return m, m.WaitForPublishEvents()
		}
		if msg.incomingEvent.Stream != m.selectedStream() {
			return m, m.WaitForPublishEvents()
		}
		m.recordPublish(time.Now())
		m.newEventStream = msg.incomingEvent.Stream
		m.newEventSubject = msg.incomingEvent.Subject
		m.newEventJobID = msg.incomingEvent.JobID
		return m, m.WaitForPublishEvents()

	case consumerSnapshotsLoadedMsg:
		m.consumerPollFetching = false
		if msg.stream != m.selectedStream() {
			return m, nil
		}

		if msg.err != nil {
			m.consumerSnapshotsErr = msg.err
			return m, nil
		}

		m.consumerSnapshotsErr = nil
		previous := make(map[string]*observe.ConsumerSnapshot, len(m.consumerSnapshots))
		for _, snapshot := range m.consumerSnapshots {
			previous[snapshot.Consumer] = snapshot
		}
		for _, snapshot := range msg.consumerSnapshots {
			m.recordConsumerSnapshot(previous[snapshot.Consumer], snapshot)
		}
		m.consumerSnapshots = msg.consumerSnapshots
		return m, nil

	case tea.KeyPressMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "r":
			if m.streamsFetching {
				return m, nil
			}
			m.streamsFetching = true
			m.streamsErr = nil
			return m, m.loadStreams()
		case "up", "k":
			if m.selected > 0 {
				m.selected--
				return m.selectStream()
			}
		case "down", "j":
			if m.selected+1 < len(m.streams) {
				m.selected++
				return m.selectStream()
			}
		}
	}
	return m, nil
}

func (m Model) selectStream() (tea.Model, tea.Cmd) {
	m.resetFlow()
	m.subjects = nil
	m.subjectsErr = nil
	m.subjectsLoading = true
	m.consumers = nil
	m.consumersErr = nil
	m.consumersLoading = true
	m.consumerSnapshots = nil
	m.consumerSnapshotsErr = nil
	stream := m.selectedStream()
	return m, tea.Batch(m.loadSubjects(stream), m.loadConsumers(stream))
}
