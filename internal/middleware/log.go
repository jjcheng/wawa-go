package middleware

import (
	"net/http/httputil"
	"time"

	"github.com/jjcheng/wawa-go/internal/cfg"
	dto_account "github.com/jjcheng/wawa-go/internal/dto/account"
	"github.com/jjcheng/wawa-go/internal/helper"
	"github.com/jjcheng/wawa-go/internal/service"

	"github.com/gin-gonic/gin"
)

// Log returns a gin.HandlerFunc (middleware) that logs requests into Azure appInsights
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
		requestUser, exist := c.Get(cfg.Default().Site.HTTPRequestUserKey)
		var appId int32
		if exist {
			app := requestUser.(*dto_account.User)
			appId = app.Id
		}
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
		requestBody, _ := httputil.DumpRequest(c.Request, false)
		logRaw(c, start, appId, string(requestBody), requestJSON, logger)
	}
}

func logRaw(c *gin.Context, startAt time.Time, appId int32, requestBody string, requestJSON string, logger *service.Logger) {
	if telemetryService := logger.TelemetryService(); telemetryService != nil {
		go func() {
			telemetryService.TrackHTTPRequest(
				c.Request.Method,
				c.Request.URL.Path,
				c.Request.URL.RawQuery,
				requestJSON,
				c.Request.UserAgent(),
				c.ClientIP(),
				requestBody,
				appId,
				c.Writer.Size(),
				time.Since(startAt),
				c.Writer.Status(),
				GetRequestID(c),
			)
		}()
	}
}
