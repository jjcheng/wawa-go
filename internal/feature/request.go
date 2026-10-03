package feature

import (
	"context"
	"encoding/json"

	"github.com/jjcheng/wawa-go/internal/dto"
	dto_account "github.com/jjcheng/wawa-go/internal/dto/account"
	"github.com/jjcheng/wawa-go/internal/service"
)

type Request[ResponseType any] interface {
	RequestHandler[ResponseType]
	RequestAPISettings
}

type RequestAPISettings interface {
	APISettings() APISettings
}

type RequestHandler[T any] interface {
	Handle(ctx context.Context, user *dto_account.User, dependencies *service.Dependencies) dto.Response[T]
}

type APIExecutor func(ctx context.Context, user *dto_account.User, dependencies *service.Dependencies, form map[string]any) dto.Response[any]

func RegisterAPIExecutor[ResponseType any, T Request[ResponseType]](summary string, request T) {
	apiIntentDescriptions.Lock()
	apiIntentDescriptions.executors[summary] = func(ctx context.Context, user *dto_account.User, dependencies *service.Dependencies, form map[string]any) dto.Response[any] {
		payload, err := json.Marshal(form)
		if err != nil {
			return dto.NewFailedResponse[any](400, "invalid feature form", err)
		}
		var requestObject T
		if err := json.Unmarshal(payload, &requestObject); err != nil {
			return dto.NewFailedResponse[any](400, "invalid feature form", err)
		}
		response := requestObject.Handle(ctx, user, dependencies)
		return dto.Response[any]{
			ResponseBase: response.ResponseBase,
			Error:        response.Error,
			Data:         response.Data,
		}
	}
	apiIntentDescriptions.Unlock()
}

func APIExecutorBySummary(summary string) (APIExecutor, bool) {
	apiIntentDescriptions.RLock()
	defer apiIntentDescriptions.RUnlock()
	executor, exists := apiIntentDescriptions.executors[summary]
	return executor, exists
}
