package shared

import "github.com/google/uuid"

type URLJob struct {
	CrawlID uuid.UUID `json:"crawlId"`
	URL     string    `json:"url"`
	Depth   int       `json:"depth"`
}
