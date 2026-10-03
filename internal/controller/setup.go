package controller

import (
	"fmt"
	"strings"

	"github.com/jjcheng/wawa-go/internal/cfg"
	dto_account "github.com/jjcheng/wawa-go/internal/dto/account"
	"github.com/jjcheng/wawa-go/internal/helper"

	"context"
	"net/http"
	"reflect"
	"time"

	"github.com/jjcheng/wawa-go/internal/dto"
	"github.com/jjcheng/wawa-go/internal/feature"
	"github.com/jjcheng/wawa-go/internal/middleware"
	"github.com/jjcheng/wawa-go/internal/service"
	"github.com/jjcheng/wawa-go/internal/types"

	"github.com/gin-gonic/gin"
)

// RegisterControllers registers all application controllers with the router
func RegisterControllers(router *gin.Engine, dependencies *service.Dependencies) {
	registerCommonRoutes(router)
	apiGenerator := feature.NewAPIGenerator()
	// router groups
	routerGroup := router.Group("")
	routerGroup.Use(middleware.Authenticate(dependencies))
	// register routes
	registerAuthController(routerGroup, dependencies, apiGenerator)
	registerAccountController(routerGroup, dependencies, apiGenerator)
	registerAdminController(routerGroup, dependencies, apiGenerator)
	registerCustomerController(routerGroup, dependencies, apiGenerator)
	registerBroadcastController(routerGroup, dependencies, apiGenerator)
	registerWAController(routerGroup, dependencies, apiGenerator)
	registerCommerceController(routerGroup, dependencies, apiGenerator)
	registerSiteController(routerGroup, dependencies, apiGenerator)
	registerPublicController(routerGroup, dependencies, apiGenerator)
	registerAIController(routerGroup, dependencies, apiGenerator)
	// generate api doc
	if cfg.Default().Site.Environment == types.EnvironmentDevelop {
		generateAPIDoc(apiGenerator, dependencies.Logger)
	}
}

func registerCommonRoutes(router *gin.Engine) {
	router.GET("/health", func(ctx *gin.Context) {
		healthResponseObject := dto.NewSuccessResponse(map[string]any{"version": cfg.Default().Site.Version, "environment": cfg.Default().Site.Environment})
		healthResponseObject.RequestId = middleware.GetRequestID(ctx)
		ctx.JSON(healthResponseObject.StatusCode, healthResponseObject)
	})
	router.NoRoute(func(ctx *gin.Context) {
		responseObject := dto.NewFailedResponse[any](http.StatusBadRequest, "route not found", nil)
		responseObject.RequestId = middleware.GetRequestID(ctx)
		ctx.AbortWithStatusJSON(responseObject.StatusCode, responseObject)
	})
	router.NoMethod(func(ctx *gin.Context) {
		responseObject := dto.NewFailedResponse[any](http.StatusMethodNotAllowed, "method not allowed", nil)
		responseObject.RequestId = middleware.GetRequestID(ctx)
		ctx.AbortWithStatusJSON(responseObject.StatusCode, responseObject)
	})
	// only serve static content in develop and staging server
	if cfg.Default().Site.Environment != types.EnvironmentProduction {
		router.Static("/www", "./www")
	}
}

func registerRoute[R any, T feature.Request[R]](server *gin.RouterGroup, dependencies *service.Dependencies, apiGenerator *feature.APIGenerator) {
	var requestType T
	if requestType.APISettings().Public {
		responseType := reflect.TypeFor[R]()
		_ = apiGenerator.AddEndpoint(requestType, responseType)
		feature.RegisterAPIExecutor(requestType.APISettings().Summary, requestType)
	}
	server.Handle(requestType.APISettings().Method, requestType.APISettings().Path, middleware.BindRequest[R, T](), func(ctx *gin.Context) {
		startAt := time.Now()
		requestObject := ctx.MustGet(cfg.Default().Site.HTTPRequestItemKey).(T)
		// try to get user
		var user *dto_account.User
		v, exist := ctx.Get(cfg.Default().Site.HTTPRequestUserKey)
		if exist {
			user = v.(*dto_account.User)
		}
		// Put gin context in request context for features that need it
		reqCtx := context.WithValue(ctx.Request.Context(), "gin", ctx)
		responseObject := requestObject.Handle(reqCtx, user, dependencies)
		endAt := time.Now()
		responseObject.StartAt = startAt
		responseObject.EndAt = endAt
		responseObject.TimeTaken = helper.GetTimeDifferenceInMS(endAt, startAt)
		responseObject.RequestId = middleware.GetRequestID(ctx)
		logResponseError(ctx, dependencies.Logger, responseObject.ResponseBase, responseObject.Error)
		if !responseObject.Success {
			ctx.AbortWithStatusJSON(responseObject.StatusCode, responseObject)
			return
		}
		if binaryData, ok := any(responseObject.Data).(interface {
			BinaryContent() ([]byte, string, string)
		}); ok {
			content, contentType, filename := binaryData.BinaryContent()
			ctx.Header("Content-Disposition", `inline; filename="`+strings.ReplaceAll(filename, `"`, "")+`"`)
			ctx.Data(responseObject.StatusCode, contentType, content)
			return
		}
		ctx.JSON(responseObject.StatusCode, responseObject)
	})
}

func logResponseError(ctx *gin.Context, logger *service.Logger, response dto.ResponseBase, err error) {
	if err == nil {
		return
	}
	logger.ErrorWithFields(err,
		"http_request_failed",
		"request_id", response.RequestId,
		"status", response.StatusCode,
		"method", ctx.Request.Method,
		"path", ctx.Request.URL.Path,
		"started_at", response.StartAt,
		"time_taken", response.TimeTaken,
	)
}

// write the api-doc.json to www/
func generateAPIDoc(generator *feature.APIGenerator, logger *service.Logger) {
	json, err := generator.GenerateJSON()
	if err != nil {
		logger.Error(fmt.Errorf("failed to generate api doc JSON: %w", err))
		return
	}
	err = helper.WriteToFile(string(json), "www/api-doc.json")
	if err != nil {
		logger.Error(fmt.Errorf("failed to write api doc file: %w", err))
		return
	}
}
