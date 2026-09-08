package dto_wa

type MessageAnalytics struct {
	PhoneNumbers   []string                    `json:"phone_numbers,omitempty"`
	Granularity    string                      `json:"granularity,omitempty"`
	DataPoints     []MessageAnalyticsDataPoint `json:"data_points"`
	TotalSent      int                         `json:"total_sent"`
	TotalDelivered int                         `json:"total_delivered"`
}

type MessageAnalyticsDataPoint struct {
	Start     int64 `json:"start"`
	End       int64 `json:"end"`
	Sent      int64 `json:"sent,omitempty"`
	Delivered int64 `json:"delivered,omitempty"`
}

// used in embedded signup to check phone number is valid
type AnalyticsPaging struct {
	Next string `json:"next,omitempty"`
}
