package service

import (
	"fmt"
	"log/slog"
	"os"
	"runtime"
	"time"

	"github.com/jjcheng/wawa-go/internal/cfg"
	"github.com/jjcheng/wawa-go/internal/types"
)

type Logger struct {
	out *slog.Logger // stdout: debug/info, JSON encoded for log-service field indexing
	err *slog.Logger // stderr: warn/error/fatal
}

func NewLogger() *Logger {
	outLevel := slog.LevelInfo
	if cfg.Default().Site.Environment == types.EnvironmentDevelop {
		outLevel = slog.LevelDebug
	}
	return &Logger{
		out: slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: outLevel})),
		err: slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn})),
	}
}

// = log.Println()
func (logger *Logger) Infoln(message string) {
	logger.out.Info(message)
}

// = log.Printf()
func (logger *Logger) Infof(message string, v ...any) {
	logger.out.Info(fmt.Sprintf(message, v...))
}

// only work in develop environment
func (logger *Logger) Debugln(message string) {
	logger.out.Debug(message)
}

// only work in develop environment
func (logger *Logger) Debugf(message string, v ...any) {
	logger.out.Debug(fmt.Sprintf(message, v...))
}

func (logger *Logger) Error(err error) {
	logger.err.Error(err.Error())
}

// function name is auto captured
func (logger *Logger) ErrorFunction(err error, values ...any) {
	funcName := getFunctionName(2)
	logger.err.Error(err.Error(), "func", funcName, "args", values)
}

func getFunctionName(skip int) string {
	pc, _, _, ok := runtime.Caller(skip)
	if !ok {
		return "unknown"
	}
	fn := runtime.FuncForPC(pc)
	if fn == nil {
		return "unknown"
	}
	return fn.Name() // returns full path, e.g. "mypkg.(*AppRepository).GetByAPIKey"
}

func (logger *Logger) Warnln(message string) {
	logger.err.Warn(message)
}

func (logger *Logger) Warnf(message string, v ...any) {
	logger.err.Warn(fmt.Sprintf(message, v...))
}

// Access logs a one-line structured summary for a completed HTTP request.
// requestJSON is only attached in the develop environment to avoid logging request payloads in prod.
func (logger *Logger) Access(method string, path string, statusCode int, duration time.Duration, userId int32, remoteAddress string, requestId string, requestJSON string) {
	attrs := []any{
		"method", method,
		"path", path,
		"status", statusCode,
		"duration", duration.String(),
		"user_id", userId,
		"ip", remoteAddress,
		"request_id", requestId,
	}
	if cfg.Default().Site.Environment == types.EnvironmentDevelop {
		attrs = append(attrs, "request_json", requestJSON)
	}
	logger.out.Info("request", attrs...)
}

// Fatal logs an unrecovered panic with the request context needed to correlate it via requestId.
func (logger *Logger) Fatal(err error, path string, query string, method string, userAgent string, remoteAddress string, requestBody string, stack string, userId int32, requestJSON string, statusCode int, requestId string) {
	logger.err.Error(err.Error(),
		"method", method,
		"path", path,
		"query", query,
		"status", statusCode,
		"user_id", userId,
		"ip", remoteAddress,
		"user_agent", userAgent,
		"request_id", requestId,
		"stack", stack,
	)
}
