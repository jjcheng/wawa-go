package feature_auth

import (
	"context"
	"net/http"

	"github.com/jjcheng/wawa-go/internal/dto"
	dto_account "github.com/jjcheng/wawa-go/internal/dto/account"
	"github.com/jjcheng/wawa-go/internal/feature"
	"github.com/jjcheng/wawa-go/internal/service"
	"github.com/jjcheng/wawa-go/internal/types"
)

type Logout struct{}

func (logout Logout) Handle(ctx context.Context, user *dto_account.User, dependencies *service.Dependencies) dto.Response[any] {
	if user == nil {
		return dto.NewFailedResponse[any](http.StatusUnauthorized, "authentication required")
	}
	if err := dependencies.UnitOfWork.AccountUserRepository().UpdateFields(ctx, user.Id, map[string]any{
		"access_token_hash":   nil,
		"access_token_expiry": nil,
	}); err != nil {
		return dto.NewFailedResponse[any](http.StatusInternalServerError, "failed to clear login session")
	}
	return dto.NewEmptyResponse(true, http.StatusOK)
}

func (Logout) APISettings() feature.APISettings {
	return feature.NewAPISettings("User logout", "Ends the authenticated user's session", types.HttpRequestTypeJSON, http.MethodPost, "/auth/v1/logout", true, false, types.APITagAuth, nil)
}
