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

func (cloudflareGetCrawlStatusResponse *CloudflareGetCrawlStatusResponse) Status() string {
	switch cloudflareGetCrawlStatusResponse.Result.Status {
	case "running":
		return "Running"
	case "cancelled_due_to_timeout":
		return "Timeout"
	case "cancelled_due_to_limits":
		return "Hit account limit"
	case "cancelled_by_user":
		return "Cancelled"
	case "errored":
		return "Error"
	case "completed":
		return "Completed"
	default:
		return cloudflareGetCrawlStatusResponse.Result.Status
	}
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
