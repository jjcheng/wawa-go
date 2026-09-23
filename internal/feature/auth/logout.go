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
		return dto.NewFailedResponse[any](http.StatusForbidden, types.ExceptionMessageForbidden, nil)
	}
	if user.Session == nil {
		return dto.NewEmptyResponse(true, http.StatusNoContent)
	}
	// get session from db, ignore error
	session, _ := dependencies.UnitOfWork.AccountSessionRepository().GetById(ctx, user.Session.Id)
	if session != nil {
		if err := dependencies.UnitOfWork.AccountSessionRepository().UpdateRevokedAt(ctx, session.Id); err == nil && dependencies.AuthCache != nil {
			dependencies.AuthCache.InvalidateSession(session.Id)
		}
	}
	return dto.NewEmptyResponse(true, http.StatusOK)
}

func (Logout) APISettings() feature.APISettings {
	return feature.NewAPISettings("User logout", "Ends the authenticated user's session", types.HttpRequestTypeJSON, http.MethodPost, "/v1/auth/logout", true, false, types.APITagAuth, nil)
}
