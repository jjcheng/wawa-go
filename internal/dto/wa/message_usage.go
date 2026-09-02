package dto_wa

type MessageAnalytics struct {
	PhoneNumbers []string                    `json:"phone_numbers,omitempty"`
	CountryCodes []string                    `json:"country_codes,omitempty"`
	Granularity  string                      `json:"granularity,omitempty"`
	DataPoints   []MessageAnalyticsDataPoint `json:"data_points"`
}

type MessageAnalyticsDataPoint struct {
	Start       int64  `json:"start"`
	End         int64  `json:"end"`
	Granularity string `json:"granularity,omitempty"`
	PhoneNumber string `json:"phone_number,omitempty"`
	Country     string `json:"country,omitempty"`
	Sent        int64  `json:"sent,omitempty"`
	Delivered   int64  `json:"delivered,omitempty"`
	Received    int64  `json:"received,omitempty"`
}

type PhoneNumberMessageAnalytics struct {
	ID                 string           `json:"id"`
	DisplayPhoneNumber string           `json:"display_phone_number,omitempty"`
	VerifiedName       string           `json:"verified_name,omitempty"`
	Analytics          MessageAnalytics `json:"analytics"`
}

type PhoneNumberMessageAnalyticsListResponse struct {
	Data   []PhoneNumberMessageAnalytics `json:"data"`
	Paging *AnalyticsPaging              `json:"paging,omitempty"`
}

type AnalyticsPaging struct {
	Next string `json:"next,omitempty"`
}
