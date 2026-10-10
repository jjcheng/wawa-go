package service

type CloudflareCrawlResponse struct {
	Success bool   `json:"success"`
	Result  string `json:"result"`
}

type CloudflareGetCrawlStatusResponse struct {
	Success bool                           `json:"success"`
	Result  CloudflareGetCrawlStatusResult `json:"result"`
}

type CloudflareGetCrawlStatusResult struct {
	Status             string                           `json:"status"`
	BrowserSecondsUsed float32                          `json:"browserSecondsUsed"`
	Total              int                              `json:"total"`
	Finished           int                              `json:"finished"`
	Records            []CloudflareGetCrawlStatusRecord `json:"records"`
	Cursor             int                              `json:"cursor"`
}

type CloudflareGetCrawlStatusRecord struct {
	URL      string                                 `json:"url"`
	Status   string                                 `json:"status"`
	Markdown string                                 `json:"markdown"`
	Metadata CloudflareGetCrawlStatusRecordMetadata `json:"metadata"`
}

type CloudflareGetCrawlStatusRecordMetadata struct {
	Status int    `json:"status"`
	Title  string `json:"title"`
	URL    string `json:"url"`
}

type CloudflareCrawlStatus string

const (
	CloudflareCrawlStatusRunning      CloudflareCrawlStatus = "RUNNING"
	CloudflareCrawlStatusTimeout      CloudflareCrawlStatus = "TIMEOUT"
	CloudflareCrawlStatusAccountLimit CloudflareCrawlStatus = "ACCOUNT LIMIT"
	CloudflareCrawlStatusCancelled    CloudflareCrawlStatus = "CANCELLED"
	CloudflareCrawlStatusError        CloudflareCrawlStatus = "ERROR"
	CloudflareCrawlStatusCompleted    CloudflareCrawlStatus = "COMPLETED"
)
