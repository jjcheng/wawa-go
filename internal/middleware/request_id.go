package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jjcheng/wawa-go/internal/cfg"
)

// RequestID returns a gin.HandlerFunc (middleware) that assigns a unique id to each
// request so access/error logs for the same request can be correlated.
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := uuid.NewString()
		c.Set(cfg.Default().Site.HTTPRequestIdKey, id)
		c.Writer.Header().Set("x-request-id", id)
		c.Next()
	}
}

// GetRequestID retrieves the request id set by RequestID, or "" if unavailable.
func GetRequestID(c *gin.Context) string {
	if v, exist := c.Get(cfg.Default().Site.HTTPRequestIdKey); exist {
		if id, ok := v.(string); ok {
			return id
		}
	}
	return ""
}
