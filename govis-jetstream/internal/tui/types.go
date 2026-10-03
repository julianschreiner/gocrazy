package tui

import (
	"time"

	"github.com/julianschreiner/govis-jetstream/internal/observe"
)

type Model struct {
	client               *observe.NatsClient
	eventHandler         *observe.EventHandler
	streams              []observe.StreamSummary
	selected             int
	streamsLoading       bool
	streamsFetching      bool
	streamsErr           error
	subjects             []string
	subjectsLoading      bool
	subjectsErr          error
	consumers            []string
	consumersLoading     bool
	consumersErr         error
	newEventStream       string
	newEventSubject      string
	newEventJobID        string
	consumerPollFetching bool
	consumerSnapshotsErr error
	consumerSnapshots    []*observe.ConsumerSnapshot
	animationNow         time.Time
	publishParticles     []flowParticle
	consumerParticles    map[string][]flowParticle
	consumerRates        map[string]float64
	publishWindowStart   time.Time
	publishWindowCount   int
	publishRate          float64
}

type streamsLoadedMsg struct {
	streams []observe.StreamSummary
	err     error
}

type subjectsLoadedMsg struct {
	stream   string
	subjects []string
	err      error
}

type consumersLoadedMsg struct {
	stream    string
	consumers []string
	err       error
}

type publishEventReceived struct {
	incomingEvent *observe.IncomingPublishEvent
	err           error
}

type refreshTickMsg struct{}
type animationTickMsg time.Time

type consumerSnapshotsLoadedMsg struct {
	stream            string
	consumerSnapshots []*observe.ConsumerSnapshot
	err               error
}
