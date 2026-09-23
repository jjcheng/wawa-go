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
	feature_wa_catalog "github.com/jjcheng/wawa-go/internal/feature/wa/catalog"
	feature_wa_message "github.com/jjcheng/wawa-go/internal/feature/wa/message"
	feature_wa_phone_number "github.com/jjcheng/wawa-go/internal/feature/wa/phone_number"
	feature_wa_product "github.com/jjcheng/wawa-go/internal/feature/wa/product"
	feature_wa_sample_template "github.com/jjcheng/wawa-go/internal/feature/wa/sample_template"
	feature_wa_template "github.com/jjcheng/wawa-go/internal/feature/wa/template"
	feature_wa_webhook "github.com/jjcheng/wawa-go/internal/feature/wa/webhook"
	"github.com/jjcheng/wawa-go/internal/helper"
	"github.com/jjcheng/wawa-go/internal/middleware"
	"github.com/jjcheng/wawa-go/internal/service"

	"github.com/gin-gonic/gin"
)

func registerWAController(routerGroup *gin.RouterGroup, dependencies *service.Dependencies, apiGenerator *feature.APIGenerator) {
	registerWebhookVerifyRoute(routerGroup, dependencies)
	registerWebhookReceiveRoute(routerGroup, dependencies)
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
	registerRoute[*service.WhatsAppPhoneNumberBusinessProfileData, feature_wa_phone_number.GetBusinessProfile](routerGroup, dependencies, apiGenerator)
	// message
	registerRoute[*dto_wa.Message, feature_wa_message.Create](routerGroup, dependencies, apiGenerator)
	registerRoute[*dto_wa.Message, feature_wa_message.Get](routerGroup, dependencies, apiGenerator)
	registerRoute[*service.AblyTokenRequest, feature_wa_message.CreateAblyToken](routerGroup, dependencies, apiGenerator)
	registerRoute[*dto.ListResponse[dto_wa.Message], feature_wa_message.List](routerGroup, dependencies, apiGenerator)
	registerRoute[*feature_wa_message.Media, feature_wa_message.GetMedia](routerGroup, dependencies, apiGenerator)
	registerMediaUploadRoute(routerGroup, dependencies, apiGenerator)
	// template
	registerRoute[*dto.ListResponse[dto_wa.Template], feature_wa_template.List](routerGroup, dependencies, apiGenerator)
	registerRoute[*dto_wa.Template, feature_wa_template.Get](routerGroup, dependencies, apiGenerator)
	registerRoute[*dto_wa.Template, feature_wa_template.Store](routerGroup, dependencies, apiGenerator)
	registerRoute[*dto_wa.Template, feature_wa_template.CreateFromSample](routerGroup, dependencies, apiGenerator)
	registerRoute[any, feature_wa_template.Delete](routerGroup, dependencies, apiGenerator)
	registerTemplateSampleUploadRoute(routerGroup, dependencies, apiGenerator)
	// catalog
	registerRoute[[]service.WhatsAppProductCatalog, feature_wa_catalog.List](routerGroup, dependencies, apiGenerator)
	registerRoute[*service.WhatsAppProductCatalog, feature_wa_catalog.Get](routerGroup, dependencies, apiGenerator)
	registerRoute[*dto.ListResponse[service.WhatsAppProduct], feature_wa_product.List](routerGroup, dependencies, apiGenerator)
	registerRoute[*dto.ListResponse[service.WhatsAppProductSet], feature_wa_catalog.ListSets](routerGroup, dependencies, apiGenerator)
	registerRoute[*dto.ListResponse[service.WhatsAppProduct], feature_wa_product.ListBySet](routerGroup, dependencies, apiGenerator)
	// sample templates
	registerRoute[[]dto_wa.SampleTemplate, feature_wa_sample_template.List](routerGroup, dependencies, apiGenerator)
	registerRoute[*dto_wa.SampleTemplate, feature_wa_sample_template.Create](routerGroup, dependencies, apiGenerator)
	// usage
	registerRoute[*dto_wa.MessageAnalytics, feature_wa_business_account.GetUsage](routerGroup, dependencies, apiGenerator)
	registerRoute[*dto_wa.MessageAnalytics, feature_wa_phone_number.GetUsage](routerGroup, dependencies, apiGenerator)
	registerRoute[[]dto_wa.TemplateAnalytics, feature_wa_template.GetUsage](routerGroup, dependencies, apiGenerator)
}

func registerWebhookVerifyRoute(routerGroup *gin.RouterGroup, dependencies *service.Dependencies) {
	routerGroup.GET(feature_wa_webhook.Verify{}.APISettings().Path, middleware.BindRequest[string, feature_wa_webhook.Verify](), func(ctx *gin.Context) {
		requestObject := ctx.MustGet(cfg.Default().Site.HTTPRequestItemKey).(feature_wa_webhook.Verify)
		responseObject := requestObject.Handle(ctx.Request.Context(), nil, dependencies)
		if !responseObject.Success {
			ctx.AbortWithStatusJSON(responseObject.StatusCode, responseObject)
			return
		}
		ctx.String(responseObject.StatusCode, responseObject.Data)
	})
}

func registerWebhookReceiveRoute(routerGroup *gin.RouterGroup, dependencies *service.Dependencies) {
	routerGroup.POST(feature_wa_webhook.Receive{}.APISettings().Path, func(ctx *gin.Context) {
		rawBody, err := ctx.GetRawData()
		if err != nil {
			ctx.AbortWithStatusJSON(http.StatusBadRequest, dto.NewFailedResponse[any](http.StatusBadRequest, "failed to read raw body", err))
			return
		}
		if !helper.VerifyWhatsAppWebhookSignature(ctx.GetHeader("X-Hub-Signature-256"), rawBody, cfg.Default().WhatsApp.AppSecret) {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, dto.NewFailedResponse[any](http.StatusUnauthorized, "invalid WhatsApp webhook signature", nil))
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
}

func registerTemplateSampleUploadRoute(routerGroup *gin.RouterGroup, dependencies *service.Dependencies, apiGenerator *feature.APIGenerator) {
	settings := feature_wa_template.UploadExample{}.APISettings()
	_ = apiGenerator.AddEndpoint(feature_wa_template.UploadExample{}, reflect.TypeFor[*service.WhatsAppTemplateHeaderSampleUploadResponse]())
	routerGroup.POST(settings.Path, func(ctx *gin.Context) {
		content, err := ctx.GetRawData()
		if err != nil {
			response := dto.NewFailedResponse[*service.WhatsAppTemplateHeaderSampleUploadResponse](http.StatusBadRequest, "failed to read sample content", err)
			ctx.AbortWithStatusJSON(response.StatusCode, response)
			return
		}
		request := feature_wa_template.UploadExample{
			Filename:    ctx.Query("filename"),
			ContentType: ctx.GetHeader("Content-Type"),
			Content:     content,
		}
		var user *dto_account.User
		if userValue, exists := ctx.Get(cfg.Default().Site.HTTPRequestUserKey); exists {
			user = userValue.(*dto_account.User)
		}
		response := request.Handle(ctx.Request.Context(), user, dependencies)
		ctx.JSON(response.StatusCode, response)
	})
}

func registerMediaUploadRoute(routerGroup *gin.RouterGroup, dependencies *service.Dependencies, apiGenerator *feature.APIGenerator) {
	settings := feature_wa_message.UploadMedia{}.APISettings()
	_ = apiGenerator.AddEndpoint(feature_wa_message.UploadMedia{}, reflect.TypeFor[*feature_wa_message.Media]())
	routerGroup.POST(settings.Path, func(ctx *gin.Context) {
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
