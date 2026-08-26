package feature

import (
	"github.com/jjcheng/wawa-go/internal/exception"
	"github.com/jjcheng/wawa-go/internal/types"
)

type APISettings struct {
	Summary     string
	Description string
	Type        types.HttpRequestType
	Method      string
	Path        string
	Auth        bool // if the endpoint need authentication
	Public      bool
	Tag         types.APITag
	Errors      []APIError
}

type APIError struct {
	StatusCode  int
	Description string
}

func NewAPISettings(summary string, description string, t types.HttpRequestType, method string, endpoint string, auth bool, public bool, tag types.APITag, errors []APIError) APISettings {
	return APISettings{
		Summary:     summary,
		Description: description,
		Type:        t,
		Method:      method,
		Path:        endpoint,
		Auth:        auth,
		Public:      public,
		Tag:         tag, // Initialize empty tag - can be set manually later
		Errors:      errors,
	}
}

func NewAPIError(ex exception.Exception) APIError {
	return APIError{
		StatusCode:  ex.StatusCode,
		Description: ex.Message,
	}
}
