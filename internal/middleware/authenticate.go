package middleware

import (
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jjcheng/wawa-go/internal/cfg"
	"github.com/jjcheng/wawa-go/internal/dto"
	dto_account "github.com/jjcheng/wawa-go/internal/dto/account"
	"github.com/jjcheng/wawa-go/internal/helper"
	"github.com/jjcheng/wawa-go/internal/service"
	"github.com/jjcheng/wawa-go/internal/types"
	"gorm.io/gorm"
)

func Authenticate(dependencies *service.Dependencies) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		userAccessToken := ctx.GetHeader(cfg.Default().Site.HTTPHeaderUserAccessTokenKey)
		if userAccessToken != "" {
			user, err := dependencies.UnitOfWork.AccountUserRepository().GetByAccessTokenHash(ctx.Request.Context(), helper.HashSHA256Hex(userAccessToken))
			if err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					responseObject := dto.NewFailedResponse[any](http.StatusUnauthorized, "invalid user")
					ctx.AbortWithStatusJSON(responseObject.StatusCode, responseObject)
					return
				}
				responseObject := dto.NewFailedResponse[any](http.StatusInternalServerError, types.ExceptionMessageInternalServerError)
				ctx.AbortWithStatusJSON(responseObject.StatusCode, responseObject)
				return
			}
			if user.AccessTokenExpiry == nil || user.AccessTokenExpiry.Before(time.Now()) {
				responseObject := dto.NewFailedResponse[any](http.StatusUnauthorized, "session expired, please login again")
				ctx.AbortWithStatusJSON(responseObject.StatusCode, responseObject)
				return
			}
			userDTO := dto_account.NewUser(*user)
			ctx.Set(cfg.Default().Site.HTTPRequestUserKey, &userDTO)
		}
		ctx.Next()
	}
}
