package middleware

import (
	"time"

	"github.com/jjcheng/wawa-go/internal/cfg"
	dto_account "github.com/jjcheng/wawa-go/internal/dto/account"
	"github.com/jjcheng/wawa-go/internal/helper"
	"github.com/jjcheng/wawa-go/internal/service"

	"github.com/gin-gonic/gin"
)

// Log returns a gin.HandlerFunc (middleware) that logs a summary line for each request
func Log(logger *service.Logger) gin.HandlerFunc {
	skipPaths := []string{"/favicon.ico"}
	return func(c *gin.Context) {
		start := time.Now()
		// some evil middlewares modify this value
		path := c.Request.URL.Path
		c.Next()
		if helper.Any(skipPaths, func(p string) bool {
			return path == p
		}) {
			return
		}
		// get user id
		requestUser, exist := c.Get(cfg.Default().Site.HTTPRequestUserKey)
		var userId int32
		if exist {
			user := requestUser.(*dto_account.User)
			userId = user.Id
		}
		// get request item
		requestItem, exist := c.Get(cfg.Default().Site.HTTPRequestItemKey)
		var requestJSON string
		if exist {
			j, err := helper.SerializeJSON(requestItem)
			if err != nil {
				requestJSON = err.Error()
			} else {
				requestJSON = *j
				if requestJSON == "{}" {
					requestJSON = ""
				}
			}
		}
		logRaw(c, start, userId, requestJSON, logger)
	}
}

func logRaw(c *gin.Context, startAt time.Time, userId int32, requestJSON string, logger *service.Logger) {
	logger.LogHTTPRequest(
		c.Request.Method,
		c.Request.URL.Path,
		c.Request.URL.RawQuery,
		c.Writer.Status(),
		time.Since(startAt),
		userId,
		c.ClientIP(),
		GetRequestID(c),
		requestJSON,
	)
}
