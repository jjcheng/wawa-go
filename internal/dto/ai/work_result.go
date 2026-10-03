package dto_ai

import (
	"strings"

	"github.com/jjcheng/wawa-go/internal/types"
)

type WorkResult struct {
	Feature string           `json:"feature"`
	Parts   []WorkResultPart `json:"parts"`
	URL     string           `json:"url"`
}

type WorkResultPart struct {
	Type    types.AIWorkResultPartType `json:"type"`
	Titles  []string                   `json:"titles,omitempty"` // for table it will be headers, for object will be the left side
	Content any                        `json:"content"`
	Input   *WorkInput                 `json:"input,omitempty"`
}

type WorkInput struct {
	Name               string              `json:"name"`
	Description        string              `json:"description"`
	Type               types.AIInputType   `json:"type"`
	DisplayType        types.AIDisplayType `json:"display_type"`
	ReferenceFieldName string              `json:"reference_field_name,omitempty"` // if coming from another source
}

func NewWorkResult(feature string, url string, parts ...WorkResultPart) WorkResult {
	workResult := WorkResult{
		Feature: feature,
		Parts:   parts,
		URL:     url,
	}
	return workResult
}

func (workResult *WorkResult) Text() string {
	var texts []string
	for _, part := range workResult.Parts {
		if part.Type == types.AIWorkResultPartTypeText {
			if text, ok := part.Content.(string); ok {
				texts = append(texts, text)
			}
		} else {
			texts = append(texts, "dynamic data")
		}
	}
	return strings.Join(texts, "\n\n")
}
