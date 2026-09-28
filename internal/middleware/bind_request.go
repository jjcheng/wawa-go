package middleware

import (
	"net/http"

	"github.com/jjcheng/wawa-go/internal/cfg"
	"github.com/jjcheng/wawa-go/internal/dto"
	"github.com/jjcheng/wawa-go/internal/exception"
	"github.com/jjcheng/wawa-go/internal/feature"
	"github.com/jjcheng/wawa-go/internal/helper"
	"github.com/jjcheng/wawa-go/internal/types"

	"github.com/gin-gonic/gin"
)

func BindRequest[R any, T feature.Request[R]]() gin.HandlerFunc {
	var requestObject T
	if requestObject.APISettings().Type == types.HttpRequestTypeNone {
		return bindNoneRequest[R, T]()
	} else if requestObject.APISettings().Type == types.HttpRequestTypeUri {
		return bindURIRequest[R, T]()
	} else if requestObject.APISettings().Type == types.HttpRequestTypeQuery {
		return bindQueryRequest[R, T]()
	} else if requestObject.APISettings().Type == types.HttpRequestTypeJSON {
		return bindJSONRequest[R, T]()
	} else if requestObject.APISettings().Type == types.HttpRequestTypeUriJSON {
		return bindURIAndJSONRequest[R, T]()
	} else {
		return bindURIAndQueryRequest[R, T]()
	}
}

func bindNoneRequest[R any, T feature.Request[R]]() gin.HandlerFunc {
	return func(context *gin.Context) {
		var requestObject T
		context.Set(cfg.Default().Site.HTTPRequestItemKey, requestObject)
		context.Next()
	}
}

// handles /intent/2
func bindURIRequest[R any, T feature.Request[R]]() gin.HandlerFunc {
	return func(context *gin.Context) {
		var requestObject T
		if err := context.ShouldBindUri(&requestObject); err != nil {
			responseObject := dto.NewInvalidInputResponse[any]([]exception.InputException{{Field: "uri", Message: err.Error()}})
			context.AbortWithStatusJSON(responseObject.StatusCode, responseObject)
			return
		}
		// validation is done at Handle()
		context.Set(cfg.Default().Site.HTTPRequestItemKey, requestObject)
		context.Next()
	}
}

// handles /intent?id=2
func bindQueryRequest[R any, T feature.Request[R]]() gin.HandlerFunc {
	return func(context *gin.Context) {
		var requestObject T
		if err := context.ShouldBindQuery(&requestObject); err != nil {
			responseObject := dto.NewInvalidInputResponse[any]([]exception.InputException{{Field: "query", Message: err.Error()}})
			context.AbortWithStatusJSON(responseObject.StatusCode, responseObject)
			return
		}
		// validation is done at Handle()
		context.Set(cfg.Default().Site.HTTPRequestItemKey, requestObject)
		context.Next()
	}
}

// handles /intent {"label": "xxx"}
func bindJSONRequest[R any, T feature.Request[R]]() gin.HandlerFunc {
	return func(context *gin.Context) {
		var requestObject T
		context.Request.Body = http.MaxBytesReader(context.Writer, context.Request.Body, helper.MaxJSONBodySizeBytes)
		contentEncodingHeader := context.GetHeader("Content-Encoding")
		switch contentEncodingHeader {
		case "gzip":
			if err := context.ShouldBindBodyWith(&requestObject, helper.GzipJSONBinding{}); err != nil {
				responseObject := dto.NewInvalidInputResponse[any]([]exception.InputException{{Field: "body", Message: err.Error()}})
				context.AbortWithStatusJSON(responseObject.StatusCode, responseObject)
				return
			}
		default:
			if err := context.ShouldBindJSON(&requestObject); err != nil {
				responseObject := dto.NewInvalidInputResponse[any]([]exception.InputException{{Field: "body", Message: err.Error()}})
				context.AbortWithStatusJSON(responseObject.StatusCode, responseObject)
				return
			}
		}
		// validation is done at Handle()
		context.Set(cfg.Default().Site.HTTPRequestItemKey, requestObject)
		context.Next()
	}
}

// handles /intents/2 {"label": "xxx"} which contains both uri and json
func bindURIAndJSONRequest[R any, T feature.Request[R]]() gin.HandlerFunc {
	return func(context *gin.Context) {
		var requestObject T
		if err := context.ShouldBindUri(&requestObject); err != nil {
			responseObject := dto.NewInvalidInputResponse[any]([]exception.InputException{{Field: "uri", Message: err.Error()}})
			context.AbortWithStatusJSON(responseObject.StatusCode, responseObject)
			return
		}
		context.Request.Body = http.MaxBytesReader(context.Writer, context.Request.Body, helper.MaxJSONBodySizeBytes)
		if err := context.ShouldBindJSON(&requestObject); err != nil {
			responseObject := dto.NewInvalidInputResponse[any]([]exception.InputException{{Field: "body", Message: err.Error()}})
			context.AbortWithStatusJSON(responseObject.StatusCode, responseObject)
			return
		}
		// validation is done at Handle()
		context.Set(cfg.Default().Site.HTTPRequestItemKey, requestObject)
		context.Next()
	}
}

// handles /intents/1?parent=2 which contains both uri and jquery
func bindURIAndQueryRequest[R any, T feature.Request[R]]() gin.HandlerFunc {
	return func(context *gin.Context) {
		var requestObject T
		if err := context.ShouldBindUri(&requestObject); err != nil {
			responseObject := dto.NewInvalidInputResponse[any]([]exception.InputException{{Field: "uri", Message: err.Error()}})
			context.AbortWithStatusJSON(responseObject.StatusCode, responseObject)
			return
		}
		if err := context.ShouldBindQuery(&requestObject); err != nil {
			responseObject := dto.NewInvalidInputResponse[any]([]exception.InputException{{Field: "query", Message: err.Error()}})
			context.AbortWithStatusJSON(responseObject.StatusCode, responseObject)
			return
		}
		// validation is done at Handle()
		context.Set(cfg.Default().Site.HTTPRequestItemKey, requestObject)
		context.Next()
	}
}
