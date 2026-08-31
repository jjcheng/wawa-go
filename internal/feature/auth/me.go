package feature_auth

import (
	"context"

	"github.com/jjcheng/wawa-go/internal/dto"
	dto_account "github.com/jjcheng/wawa-go/internal/dto/account"
	"github.com/jjcheng/wawa-go/internal/feature"
	"github.com/jjcheng/wawa-go/internal/service"
	"github.com/jjcheng/wawa-go/internal/types"
)

type Me struct{}

func (me Me) Handle(_ context.Context, user *dto_account.User, _ *service.Dependencies) dto.Response[*dto_account.User] {
	return dto.NewSuccessResponse(user)
}

func (Me) APISettings() feature.APISettings {
	return feature.NewAPISettings("Get current user", "Returns the authenticated user", types.HttpRequestTypeNone, "GET", "/v1/auth/me", true, false, types.APITagAuth, nil)
}
