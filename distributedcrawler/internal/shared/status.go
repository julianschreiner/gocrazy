package shared

import "github.com/google/uuid"

type JobStatus string

const (
	StatusPending      JobStatus = "pending"
	StatusRunning      JobStatus = "running"
	StatusDone         JobStatus = "done"
	StatusFailed       JobStatus = "failed"
	StatusDuplicate    JobStatus = "duplicate"
	StatusRetryPending JobStatus = "retry_pending"
)

type CrawlStatus struct {
	CrawlID uuid.UUID `json:"crawlId"`
	Status  JobStatus `json:"status"`
}
