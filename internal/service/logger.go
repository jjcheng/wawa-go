package service

import (
	"fmt"
	"log"
	"os"
	"runtime"
	"runtime/debug"

	"github.com/jjcheng/wawa-go/internal/cfg"
	"github.com/jjcheng/wawa-go/internal/types"
)

type Logger struct {
	debugLogger      *log.Logger
	infoLogger       *log.Logger
	errorLogger      *log.Logger
	warnLogger       *log.Logger
	fatalLogger      *log.Logger
	telemetryService *Telemetry
}

func NewLogger() *Logger {
	return &Logger{
		telemetryService: NewTelemetry(),
		debugLogger:      log.New(os.Stdout, "[DEBUG] ", log.LstdFlags),
		infoLogger:       log.New(os.Stdout, "[INFO] ", log.LstdFlags),
		errorLogger:      log.New(os.Stderr, "[ERROR] ", log.LstdFlags),
		warnLogger:       log.New(os.Stderr, "[WARNING] ", log.LstdFlags),
		fatalLogger:      log.New(os.Stderr, "[FATAL] ", log.LstdFlags),
	}
}

func (logger *Logger) TelemetryService() *Telemetry {
	return logger.telemetryService
}

// = log.Println()
func (logger *Logger) Infoln(message string) {
	//logger.infoLogger.Println(message)
	log.Println(message)
	if logger.telemetryService != nil {
		dic := logger.newPropertiesDic(message)
		logger.telemetryService.TrackEvent("INFO", dic, nil)
	}
}

// = log.Printf()
func (logger *Logger) Infof(message string, v ...any) {
	//logger.infoLogger.Printf(message, v...)
	log.Printf(message, v...)
	if logger.telemetryService != nil {
		dic := logger.newPropertiesDic(fmt.Sprintf(message, v...))
		logger.telemetryService.TrackEvent("INFO", dic, nil)
	}
}

// only work in develop environment
// = log.Println("[DEBUG] ")
func (logger *Logger) Debugln(message string) {
	if cfg.Default().Site.Environment != types.EnvironmentDevelop {
		return
	}
	logger.debugLogger.Println(message)
}

// only work in develop environment
// = log.Printf("[DEBUG] ")
func (logger *Logger) Debugf(message string, v ...any) {
	if cfg.Default().Site.Environment != types.EnvironmentDevelop {
		return
	}
	logger.debugLogger.Printf(message, v...)
}

func (logger *Logger) Error(err error) {
	logger.errorLogger.Println(err.Error())
	if logger.telemetryService != nil {
		dic := logger.newPropertiesDic(err.Error())
		logger.telemetryService.TrackError(err, dic)
	}
}

// function name is auto captured
func (logger *Logger) ErrorFunction(err error, values ...any) {
	funcName := getFunctionName(2)
	var message string
	if len(values) > 0 {
		message = fmt.Sprintf("%s(%v)", funcName, values)
	} else {
		message = funcName
	}
	logger.errorLogger.Printf("\nERROR: %s\nFUNC: %s\n", err.Error(), message)
	if logger.telemetryService != nil {
		dic := logger.newPropertiesDic(message)
		dic["function"] = funcName
		dic["args"] = fmt.Sprintf("%v", values)
		dic["stack"] = string(debug.Stack())
		logger.telemetryService.TrackError(err, dic)
	}
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

// = log.Println("[WARNING] ")
func (logger *Logger) Warnln(message string) {
	logger.warnLogger.Println(message)
	if logger.telemetryService != nil {
		dic := logger.newPropertiesDic(message)
		logger.telemetryService.TrackEvent("WARNING", dic, nil)
	}
}

// = log.Printf("[WARNING] ")
func (logger *Logger) Warnf(message string, v ...any) {
	logger.warnLogger.Printf(message, v...)
	if logger.telemetryService != nil {
		dic := logger.newPropertiesDic(fmt.Sprintf(message, v...))
		logger.telemetryService.TrackEvent("WARNING", dic, nil)
	}
}

// = log.Println("[FATAL] ")
func (logger *Logger) Fatal(err error, path string, query string, method string, userAgent string, remoteAddress string, requestBody string, stack string, appId int32, requestJSON string, statusCode int) {
	if logger.telemetryService != nil {
		dic := logger.newPropertiesDic(err.Error())
		dic["appId"] = fmt.Sprint(appId)
		dic["method"] = method
		dic["path"] = path
		dic["query"] = query
		dic["userAgent"] = userAgent
		dic["remoteAddress"] = remoteAddress
		dic["requestBody"] = requestBody
		dic["requestJSON"] = requestJSON
		dic["stack"] = stack
		dic["status"] = fmt.Sprint(statusCode)
		logger.telemetryService.TrackError(err, dic)
	}
}

// will be sent to appInsights
func (logger *Logger) newPropertiesDic(message string) map[string]string {
	dic := map[string]string{
		"environment": string(cfg.Default().Site.Environment),
		"message":     message,
	}
	return dic
}
