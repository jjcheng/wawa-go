package controller

import (
	"net/http"
	"reflect"

	"github.com/jjcheng/wawa-go/internal/cfg"
	"github.com/jjcheng/wawa-go/internal/dto"
	dto_account "github.com/jjcheng/wawa-go/internal/dto/account"
	dto_wa "github.com/jjcheng/wawa-go/internal/dto/wa"
	"github.com/jjcheng/wawa-go/internal/feature"
	feature_wa_account "github.com/jjcheng/wawa-go/internal/feature/wa/account"
	feature_wa_business_account "github.com/jjcheng/wawa-go/internal/feature/wa/business_account"
	feature_wa_message "github.com/jjcheng/wawa-go/internal/feature/wa/message"
	feature_wa_phone_number "github.com/jjcheng/wawa-go/internal/feature/wa/phone_number"
	feature_wa_sample_template "github.com/jjcheng/wawa-go/internal/feature/wa/sample_template"
	feature_wa_template "github.com/jjcheng/wawa-go/internal/feature/wa/template"
	feature_wa_webhook "github.com/jjcheng/wawa-go/internal/feature/wa/webhook"
	"github.com/jjcheng/wawa-go/internal/helper"
	"github.com/jjcheng/wawa-go/internal/middleware"
	"github.com/jjcheng/wawa-go/internal/service"

	"github.com/gin-gonic/gin"
)

func registerWAController(routerGroup *gin.RouterGroup, dependencies *service.Dependencies, apiGenerator *feature.APIGenerator) {
	// verify endpoint during configuration
	routerGroup.GET(feature_wa_webhook.Verify{}.APISettings().Path, middleware.BindRequest[string, feature_wa_webhook.Verify](), func(ctx *gin.Context) {
		requestObject := ctx.MustGet(cfg.Default().Site.HTTPRequestItemKey).(feature_wa_webhook.Verify)
		responseObject := requestObject.Handle(ctx.Request.Context(), nil, dependencies)
		if !responseObject.Success {
			ctx.AbortWithStatusJSON(responseObject.StatusCode, responseObject)
			return
		}
		ctx.String(responseObject.StatusCode, responseObject.Data)
	})
	// webhook to receive incoming content, can be messages, status, history etc...
	routerGroup.POST(feature_wa_webhook.Receive{}.APISettings().Path, func(ctx *gin.Context) {
		rawBody, err := ctx.GetRawData()
		if err != nil {
			ctx.AbortWithStatusJSON(http.StatusBadRequest, dto.NewFailedResponse[any](http.StatusUnauthorized, "failed to read raw body"))
			return
		}
		if !helper.VerifyWhatsAppWebhookSignature(ctx.GetHeader("X-Hub-Signature-256"), rawBody, cfg.Default().WhatsApp.AppSecret) {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, dto.NewFailedResponse[any](http.StatusUnauthorized, "invalid WhatsApp webhook signature"))
			return
		}
		requestObject := feature_wa_webhook.Receive{RawBody: string(rawBody)}
		responseObject := requestObject.Handle(ctx.Request.Context(), nil, dependencies)
		if !responseObject.Success {
			ctx.AbortWithStatusJSON(responseObject.StatusCode, responseObject)
			return
		}
		ctx.JSON(responseObject.StatusCode, responseObject)
	})
	// embedded signup
	registerRoute[*dto_account.User, feature_wa_account.EmbeddedSignup](routerGroup, dependencies, apiGenerator)
	// business account
	registerRoute[*dto_wa.BusinessAccount, feature_wa_business_account.Get](routerGroup, dependencies, apiGenerator)
	registerRoute[*dto_wa.BusinessAccount, feature_wa_business_account.Update](routerGroup, dependencies, apiGenerator)
	// phone number
	registerRoute[*service.WhatsAppPhoneNumberDetailsResponse, feature_wa_phone_number.Get](routerGroup, dependencies, apiGenerator)
	registerRoute[any, feature_wa_phone_number.Disconnect](routerGroup, dependencies, apiGenerator)
	registerRoute[any, feature_wa_phone_number.Reconnect](routerGroup, dependencies, apiGenerator)
	registerRoute[*dto.ListResponse[dto_wa.PhoneNumber], feature_wa_phone_number.List](routerGroup, dependencies, apiGenerator)
	registerRoute[any, feature_wa_phone_number.Delete](routerGroup, dependencies, apiGenerator)
	// message
	registerRoute[*dto_wa.Message, feature_wa_message.Create](routerGroup, dependencies, apiGenerator)
	registerRoute[*dto_wa.Message, feature_wa_message.Get](routerGroup, dependencies, apiGenerator)
	registerRoute[*service.AblyTokenRequest, feature_wa_message.CreateChatToken](routerGroup, dependencies, apiGenerator)
	registerRoute[*dto.ListResponse[dto_wa.Message], feature_wa_message.List](routerGroup, dependencies, apiGenerator)
	registerRoute[*feature_wa_message.Media, feature_wa_message.GetMedia](routerGroup, dependencies, apiGenerator)
	registerWAMediaUploadRoute(routerGroup, dependencies, apiGenerator)
	// template
	registerRoute[*dto.ListResponse[dto_wa.Template], feature_wa_template.List](routerGroup, dependencies, apiGenerator)
	registerRoute[*dto_wa.Template, feature_wa_template.Create](routerGroup, dependencies, apiGenerator)
	registerRoute[*dto_wa.Template, feature_wa_template.CreateFromSample](routerGroup, dependencies, apiGenerator)
	registerRoute[any, feature_wa_template.Delete](routerGroup, dependencies, apiGenerator)
	// sample templates
	registerRoute[[]dto_wa.SampleTemplate, feature_wa_sample_template.List](routerGroup, dependencies, apiGenerator)
	registerRoute[*dto_wa.SampleTemplate, feature_wa_sample_template.Create](routerGroup, dependencies, apiGenerator)
	// usage
	registerRoute[*dto_wa.MessageAnalytics, feature_wa_business_account.GetUsage](routerGroup, dependencies, apiGenerator)
	registerRoute[*dto_wa.MessageAnalytics, feature_wa_phone_number.GetUsage](routerGroup, dependencies, apiGenerator)
	registerRoute[[]dto_wa.TemplateAnalytics, feature_wa_template.GetUsage](routerGroup, dependencies, apiGenerator)
}

func registerWAMediaUploadRoute(routerGroup *gin.RouterGroup, dependencies *service.Dependencies, apiGenerator *feature.APIGenerator) {
	_ = apiGenerator.AddEndpoint(feature_wa_message.UploadMedia{}, reflect.TypeFor[*feature_wa_message.Media]())
	routerGroup.POST("/v1/wa/media", func(ctx *gin.Context) {
		content, err := ctx.GetRawData()
		if err != nil {
			response := feature_wa_message.UploadMediaRequestError(err)
			ctx.AbortWithStatusJSON(response.StatusCode, response)
			return
		}
		request := feature_wa_message.NewUploadMedia(ctx.Query("filename"), ctx.GetHeader("Content-Type"), content)
		request.ToMeta = ctx.Query("to_meta") == "true"
		var user *dto_account.User
		if userValue, exists := ctx.Get(cfg.Default().Site.HTTPRequestUserKey); exists {
			user = userValue.(*dto_account.User)
		}
		response := request.Handle(ctx.Request.Context(), user, dependencies)
		ctx.JSON(response.StatusCode, response)
	})
}
