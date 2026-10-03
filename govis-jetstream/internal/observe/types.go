package observe

import "time"

// StreamSnapshot, ConsumerSnapshot, Event

type (
	StreamSummary struct {
		Name         string
		CreatedAt    time.Time
		NumConsumers int
		NumMessages  uint64
		FirstSeq     uint64
		LastSeq      uint64
		MaxConsumers int
		MaxMessages  int64
	}
	ConfiguredStreamSubjectPatterns struct {
		Stream         string
		SubjectPattern []string
	}
	ConsumerSnapshot struct {
		Stream               string
		Consumer             string
		ObservedAt           time.Time
		DeliveredConsumerSeq uint64
		DeliveredStreamSeq   uint64
		AckFloorConsumerSeq  uint64
		AckFloorStreamSeq    uint64
		NumPending           uint64
		NumAckPending        int
	}
	// TODO might need to add eent types
)
