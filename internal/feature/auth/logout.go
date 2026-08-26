package feature_auth

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jjcheng/wawa-go/internal/cfg"
	"github.com/jjcheng/wawa-go/internal/dto"
	dto_account "github.com/jjcheng/wawa-go/internal/dto/account"
	"github.com/jjcheng/wawa-go/internal/feature"
	"github.com/jjcheng/wawa-go/internal/helper"
	"github.com/jjcheng/wawa-go/internal/service"
	"github.com/jjcheng/wawa-go/internal/types"
)

type Logout struct{}

func (logout Logout) Handle(ctx context.Context, _ *dto_account.User, dependencies *service.Dependencies) dto.Response[any] {
	ginContext, ok := ctx.Value("gin").(*gin.Context)
	if !ok {
		return dto.NewFailedResponse[any](http.StatusInternalServerError, "failed to clear login session")
	}
	if cookie, err := ginContext.Request.Cookie(cfg.Default().Site.SessionCookieName); err == nil && cookie.Value != "" {
		if err := dependencies.UnitOfWork.AccountSessionRepository().RevokeByTokenHash(ctx, helper.HashSHA256Hex(cookie.Value)); err != nil {
			return dto.NewFailedResponse[any](http.StatusInternalServerError, "failed to clear login session")
		}
	}
	secure := cfg.Default().Site.Environment != types.EnvironmentDevelop
	ginContext.SetSameSite(http.SameSiteLaxMode)
	ginContext.SetCookie(cfg.Default().Site.SessionCookieName, "", -1, "/", "", secure, true)
	return dto.NewEmptyResponse(true, http.StatusOK)
}

func (Logout) APISettings() feature.APISettings {
	return feature.NewAPISettings("User logout", "Ends the authenticated user's session", types.HttpRequestTypeJSON, http.MethodPost, "/auth/v1/logout", true, false, types.APITagAuth, nil)
}
