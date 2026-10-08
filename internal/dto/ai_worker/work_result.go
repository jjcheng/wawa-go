package dto_ai_worker

import (
	"github.com/jjcheng/wawa-go/internal/types"
)

type WorkResult struct {
	Feature string           `json:"feature"`
	Parts   []WorkResultPart `json:"parts"`
	URL     string           `json:"url"`
}

type WorkInput struct {
	Name               string                    `json:"name"`
	Description        string                    `json:"description"`
	Type               types.AIWorkerInputType   `json:"type"`
	DisplayType        types.AIWorkerDisplayType `json:"display_type"`
	Values             []string                  `json:"values,omitempty"` // if type is select, got from types.xxx
	Example            string                    `json:"example,omitempty"`
	ReferenceFieldName string                    `json:"reference_field_name,omitempty"` // if coming from another source
}

type WorkResultPart struct {
	FieldNames []string   `json:"field_names,omitempty"` // for list to know the field names
	Titles     []string   `json:"titles,omitempty"`      // for table it will be headers, for object will be the left side
	Content    any        `json:"content"`               // can be text, object or array
	Input      *WorkInput `json:"input,omitempty"`       // if this is used in a form
	Color      string     `json:"color,omitempty"`       // to represent success, failure
}

func NewWorkResult(feature string, url string, parts ...WorkResultPart) WorkResult {
	workResult := WorkResult{
		Feature: feature,
		Parts:   parts,
		URL:     url,
	}
	return workResult
}
