package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jjcheng/wawa-go/internal/cfg"
	"github.com/jjcheng/wawa-go/internal/dto"
	dto_account "github.com/jjcheng/wawa-go/internal/dto/account"
	"github.com/jjcheng/wawa-go/internal/helper"
	"github.com/jjcheng/wawa-go/internal/service"
)

func Authenticate(dependencies *service.Dependencies) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		cookie, err := ctx.Request.Cookie(cfg.Default().Site.SessionCookieName)
		if err != nil || cookie.Value == "" {
			responseObject := dto.NewFailedResponse[any](http.StatusUnauthorized, "authentication required")
			ctx.AbortWithStatusJSON(responseObject.StatusCode, responseObject)
			return
		}
		user, lookupErr, notFound := dependencies.UnitOfWork.AccountSessionRepository().GetUserByTokenHash(ctx.Request.Context(), helper.HashSHA256Hex(cookie.Value))
		if lookupErr != nil || notFound || user == nil {
			responseObject := dto.NewFailedResponse[any](http.StatusUnauthorized, "invalid or expired session")
			ctx.AbortWithStatusJSON(responseObject.StatusCode, responseObject)
			return
		}
		userDTO := dto_account.NewUser(*user)
		ctx.Set(cfg.Default().Site.HTTPRequestUserKey, &userDTO)
		ctx.Next()
	}
}
