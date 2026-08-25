package feature

import (
	"context"

	"github.com/jjcheng/wawa-go/internal/dto"
	dto_account "github.com/jjcheng/wawa-go/internal/dto/account"
	"github.com/jjcheng/wawa-go/internal/service"
)

type RequestObject[ResponseType any] interface {
	RequestObjectHandler[ResponseType]
	APIObject
}

type APIObject interface {
	APISettings() APISettings
}

type RequestObjectHandler[T any] interface {
	Handle(ctx context.Context, user *dto_account.User, dependencies *service.Dependencies) dto.Response[T]
}
