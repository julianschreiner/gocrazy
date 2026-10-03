package shared

import "github.com/google/uuid"

type CrawlRequest struct {
	StartURL       string `json:"startUrl"`
	MaxDepth       int    `json:"maxDepth"`
	MaxPages       int    `json:"maxPages"`
	SameDomainOnly bool   `json:"sameDomainOnly"`
}

// CrawlStartRequest is the durable command sent to the orchestrator.
type CrawlStartRequest struct {
	CrawlID uuid.UUID    `json:"crawlId"`
	Request CrawlRequest `json:"request"`
}
