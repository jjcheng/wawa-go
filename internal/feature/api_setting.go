package feature

import (
	"context"

	"github.com/jjcheng/wawa-go/internal/dto"
	dto_account "github.com/jjcheng/wawa-go/internal/dto/account"
	dto_ai "github.com/jjcheng/wawa-go/internal/dto/ai"
	"github.com/jjcheng/wawa-go/internal/exception"
	"github.com/jjcheng/wawa-go/internal/service"
	"github.com/jjcheng/wawa-go/internal/types"
)

type APISettings struct {
	Summary         string
	Description     string
	Type            types.HttpRequestType
	Method          string
	Path            string
	Auth            bool
	Public          bool
	Tag             types.APITag
	Errors          []APIError
	BodyContentType string
	AIWorker        *AIWorker
}

type APIError struct {
	StatusCode  int
	Description string
}

func NewAPISettings(summary string, description string, t types.HttpRequestType, method string, endpoint string, auth bool, public bool, tag types.APITag, errors []APIError, aiWorkers ...*AIWorker) APISettings {
	var aiWorker *AIWorker
	if len(aiWorkers) > 0 {
		aiWorker = aiWorkers[0]
	}
	return APISettings{
		Summary:         summary,
		Description:     description,
		Type:            t,
		Method:          method,
		Path:            endpoint,
		Auth:            auth,
		Public:          public,
		Tag:             tag,
		Errors:          errors,
		BodyContentType: "application/json",
		AIWorker:        aiWorker,
	}
}

func NewBinaryAPISettings(summary string, description string, t types.HttpRequestType, method string, endpoint string, auth bool, public bool, tag types.APITag, errors []APIError, aiWorkers ...*AIWorker) APISettings {
	settings := NewAPISettings(summary, description, t, method, endpoint, auth, public, tag, errors, aiWorkers...)
	settings.BodyContentType = "application/octet-stream"
	return settings
}

func NewAPIError(ex exception.Exception) APIError {
	return APIError{
		StatusCode:  ex.StatusCode,
		Description: ex.Message,
	}
}

// indicates this feature can be exeucted by AI worker
type AIWorker struct {
	// e.g. to create a customer, go to Chats page, click Add customer, key in customer details and save; alternatively, you also give me your customer information, I can do it for you
	HowToMessage string
	// indicate this feature can be exexuted completely by AI. If true, will make API calls directly, if not, will show user the HotToMessage
	Executable bool
	// if executable is true, tell user what are the prerequisites like users, phone numbers etc...
	Requires []AIWorkerRequire
	// determins the response type if success
	ReturnType types.AIWorkerReturnType
	// if returnType is text, will return this; otherwise will return the data generated
	ReturnText string
	URL        string
}

func NewAIWorker(executable bool, howToMessage string, returnType types.AIWorkerReturnType, returnText string, url string, requires ...AIWorkerRequire) *AIWorker {
	return &AIWorker{
		Executable:   executable,
		HowToMessage: howToMessage,
		Requires:     requires,
		ReturnType:   returnType,
		ReturnText:   returnText,
		URL:          url,
	}
}

type AIWorkerRequire struct {
	Title   string
	Type    types.AIWorkResultPartType
	Handler func(ctx context.Context, user *dto_account.User, dependencies *service.Dependencies) dto.Response[any]
	Input   dto_ai.WorkInput
}

func NewAIWorkerRequire[T any](title string, request Request[T], input dto_ai.WorkInput) AIWorkerRequire {
	return AIWorkerRequire{
		Title: title,
		Handler: func(ctx context.Context, user *dto_account.User, dependencies *service.Dependencies) dto.Response[any] {
			response := request.Handle(ctx, user, dependencies)
			return dto.Response[any]{
				ResponseBase: response.ResponseBase,
				Error:        response.Error,
				Data:         response.Data,
			}
		},
		Input: input,
	}
}
