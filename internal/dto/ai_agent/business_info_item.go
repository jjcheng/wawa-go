package dto_ai_agent

import (
	"strings"

	"github.com/jjcheng/wawa-go/internal/exception"
)

type BusinessInfoItem struct {
	Title       string `json:"title" val:"required" description:"title of the item"`
	Description string `json:"description" val:"required" description:"description of the item"`
}

func (businessInfoItem *BusinessInfoItem) Validate(index int) []exception.InputException {
	var errors []exception.InputException
	businessInfoItem.Title = strings.TrimSpace(businessInfoItem.Title)
	if businessInfoItem.Title == "" {
		errors = append(errors, exception.NewInputException("title", "missing title"))
	}
	businessInfoItem.Description = strings.TrimSpace(businessInfoItem.Description)
	if businessInfoItem.Description == "" {
		errors = append(errors, exception.NewInputException("description", "missing descriptions"))
	}
	return errors
}

func (businessInfoItem BusinessInfoItem) Payload() map[string]any {
	return map[string]any{
		"title":       businessInfoItem.Title,
		"description": businessInfoItem.Description,
	}
}
