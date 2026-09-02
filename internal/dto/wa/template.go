package dto_wa

type Template struct {
	ID               string           `json:"id"`
	MetaWABAId       string           `json:"meta_waba_id"`
	Name             string           `json:"name"`
	Status           string           `json:"status"`
	Category         string           `json:"category"`
	Language         string           `json:"language"`
	ParameterFormat  string           `json:"parameter_format,omitempty"`
	Components       []map[string]any `json:"components,omitempty"`
	QualityScore     map[string]any   `json:"quality_score,omitempty"`
	RejectedReason   string           `json:"rejected_reason,omitempty"`
	PreviousCategory string           `json:"previous_category,omitempty"`
}

type TemplateListResponse struct {
	Data   []Template      `json:"data"`
	Paging *TemplatePaging `json:"paging,omitempty"`
}

type TemplatePaging struct {
	Previous string                 `json:"previous"`
	Next     string                 `json:"next"`
	Cursors  *TemplatePagingCursors `json:"cursors,omitempty"`
}

type TemplatePagingCursors struct {
	Before string `json:"before,omitempty"`
	After  string `json:"after,omitempty"`
}

type TemplateAnalytics struct {
	WABATimezone string                       `json:"waba_timezone,omitempty"`
	Granularity  string                       `json:"granularity,omitempty"`
	ProductType  string                       `json:"product_type,omitempty"`
	DataPoints   []TemplateAnalyticsDataPoint `json:"data_points"`
}

type TemplateAnalyticsDataPoint struct {
	TemplateID string                  `json:"template_id"`
	Start      int64                   `json:"start"`
	End        int64                   `json:"end"`
	Sent       int64                   `json:"sent,omitempty"`
	Delivered  int64                   `json:"delivered,omitempty"`
	Read       int64                   `json:"read,omitempty"`
	Cost       []TemplateAnalyticsCost `json:"cost,omitempty"`
}

type TemplateAnalyticsCost struct {
	Type  string  `json:"type"`
	Value float64 `json:"value"`
}

type TemplateAnalyticsListResponse struct {
	Data   []TemplateAnalytics `json:"data"`
	Paging *TemplatePaging     `json:"paging,omitempty"`
}
