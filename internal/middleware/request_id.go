package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jjcheng/wawa-go/internal/cfg"
)

const requestIDHeader = "X-Wawa-Request-ID"

func RequestID() gin.HandlerFunc {
	return func(context *gin.Context) {
		requestID := uuid.NewString()
		context.Set(cfg.Default().Site.HTTPRequestIdKey, requestID)
		context.Writer.Header().Set(requestIDHeader, requestID)
		context.Next()
	}
}

func GetRequestID(context *gin.Context) string {
	if value, exists := context.Get(cfg.Default().Site.HTTPRequestIdKey); exists {
		if requestID, ok := value.(string); ok {
			return requestID
		}
	}
	return ""
}
