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
	Next string `json:"next,omitempty"`
}
