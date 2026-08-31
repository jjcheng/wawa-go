package feature

import (
	"context"

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
