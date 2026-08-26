package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jjcheng/wawa-go/internal/cfg"
	"github.com/jjcheng/wawa-go/internal/dto"
)

func CSRF() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		switch ctx.Request.Method {
		case http.MethodGet, http.MethodHead, http.MethodOptions:
			ctx.Next()
			return
		}
		origin := ctx.GetHeader("Origin")
		allowedOrigin := cfg.Default().Site.PortalOrigin
		if allowedOrigin == "" {
			allowedOrigin = "http://localhost:3000"
		}
		if origin != allowedOrigin {
			responseObject := dto.NewFailedResponse[any](http.StatusForbidden, "invalid request origin")
			ctx.AbortWithStatusJSON(responseObject.StatusCode, responseObject)
			return
		}
		ctx.Next()
	}
}
