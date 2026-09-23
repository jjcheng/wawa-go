package middleware

import (
	"fmt"
	"net"
	"net/http"
	"net/http/httputil"
	"os"
	"runtime/debug"
	"strings"

	"github.com/jjcheng/wawa-go/internal/cfg"
	dto_account "github.com/jjcheng/wawa-go/internal/dto/account"
	"github.com/jjcheng/wawa-go/internal/exception"
	"github.com/jjcheng/wawa-go/internal/helper"
	"github.com/jjcheng/wawa-go/internal/service"
	"github.com/jjcheng/wawa-go/internal/types"

	"github.com/gin-gonic/gin"
)

func defaultHandleRecovery(context *gin.Context, err any) {
	ex := exception.NewException(types.ExceptionTypeInternalServer)
	context.AbortWithStatusJSON(http.StatusInternalServerError, ex)
}

func Recovery(logger *service.Logger) gin.HandlerFunc {
	return CustomRecovery(defaultHandleRecovery, logger)
}

// CustomRecovery returns a gin.HandlerFunc (middleware) with a custom recovery handler
// that recovers from any panics and logs requests.
// All errors are logged using Error().
// The stack info is easy to find where the error occurs but can be verbose.
func CustomRecovery(recovery gin.RecoveryFunc, logger *service.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				panicErr := errorFromPanic(err)
				// Check for a broken connection, as it is not really a
				// condition that warrants a panic stack trace.
				var brokenPipe bool
				if ne, ok := panicErr.(*net.OpError); ok {
					if se, ok := ne.Err.(*os.SyscallError); ok {
						if strings.Contains(strings.ToLower(se.Error()), "broken pipe") || strings.Contains(strings.ToLower(se.Error()), "connection reset by peer") {
							brokenPipe = true
						}
					}
				}
				if brokenPipe {
					// If the connection is dead, we can't write a status to it.
					c.Error(panicErr) // nolint: errcheck
					c.Abort()
					return
				}
				httpRequest, _ := httputil.DumpRequest(c.Request, false)
				requestUser, exist := c.Get(cfg.Default().Site.HTTPRequestUserKey)
				var userId int32
				if exist {
					user := requestUser.(*dto_account.User)
					userId = user.Id
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
				logger.Fatal(panicErr, c.Request.URL.Path, c.Request.URL.RawQuery, c.Request.Method, c.Request.UserAgent(), c.ClientIP(), string(httpRequest), string(debug.Stack()), userId, requestJSON, http.StatusInternalServerError, GetRequestID(c))
				recovery(c, err)
			}
		}()
		c.Next()
	}
}

func errorFromPanic(err any) error {
	if err == nil {
		return fmt.Errorf("panic: <nil>")
	}
	if e, ok := err.(error); ok {
		return e
	}
	return fmt.Errorf("panic: %v", err)
}
