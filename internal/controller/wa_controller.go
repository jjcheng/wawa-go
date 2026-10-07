package controller

import (
	"io"
	"net/http"
	"reflect"
	"strings"
	"time"

	"github.com/jjcheng/wawa-go/internal/cfg"
	"github.com/jjcheng/wawa-go/internal/dto"
	dto_account "github.com/jjcheng/wawa-go/internal/dto/account"
	dto_wa "github.com/jjcheng/wawa-go/internal/dto/wa"
	"github.com/jjcheng/wawa-go/internal/exception"
	"github.com/jjcheng/wawa-go/internal/feature"
	feature_wa_account "github.com/jjcheng/wawa-go/internal/feature/wa/account"
	feature_wa_business_account "github.com/jjcheng/wawa-go/internal/feature/wa/business_account"
	feature_wa_business_agent "github.com/jjcheng/wawa-go/internal/feature/wa/business_agent"
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
	"github.com/jjcheng/wawa-go/internal/types"

	"github.com/gin-gonic/gin"
)

func registerWAController(routerGroup *gin.RouterGroup, dependencies *service.Dependencies, apiGenerator *feature.APIGenerator) {
	registerWebhookVerifyRoute(routerGroup, dependencies)
	registerWebhookReceiveRoute(routerGroup, dependencies)
	// embedded signup
	registerRoute[*feature_wa_account.EmbeddedSignupResult, feature_wa_account.EmbeddedSignup](routerGroup, dependencies, apiGenerator)
	// business account
	registerRoute[*dto_wa.BusinessAccount, feature_wa_business_account.Get](routerGroup, dependencies, apiGenerator)
	registerRoute[*dto_wa.BusinessAccount, feature_wa_business_account.Update](routerGroup, dependencies, apiGenerator)
	// phone number
	registerRoute[*service.WhatsAppPhoneNumberDetailsResponse, feature_wa_phone_number.Get](routerGroup, dependencies, apiGenerator)
	registerRoute[*dto_wa.PhoneNumber, feature_wa_phone_number.GetLocal](routerGroup, dependencies, apiGenerator)
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
	registerRoute[any, feature_wa_message.MarkRead](routerGroup, dependencies, apiGenerator)
	registerRoute[any, feature_wa_message.StartTyping](routerGroup, dependencies, apiGenerator)
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
	// business agent
	registerRoute[any, feature_wa_business_agent.TurnOn](routerGroup, dependencies, apiGenerator)
	registerRoute[*feature_wa_business_agent.CheckEligibilityResult, feature_wa_business_agent.CheckEligibility](routerGroup, dependencies, apiGenerator)
	registerRoute[*feature_wa_business_agent.OnboardResult, feature_wa_business_agent.Onboard](routerGroup, dependencies, apiGenerator)
	registerRoute[any, feature_wa_business_agent.Offboard](routerGroup, dependencies, apiGenerator)
	registerRoute[[]service.AgentBudget, feature_wa_business_agent.ListBudgets](routerGroup, dependencies, apiGenerator)
	registerRoute[[]service.AgentBudget, feature_wa_business_agent.StoreBudgets](routerGroup, dependencies, apiGenerator)
	registerRoute[*service.AgentPhoneNumberBusinessInfo, feature_wa_business_agent.GetBusinessInfo](routerGroup, dependencies, apiGenerator)
	registerRoute[any, feature_wa_business_agent.UpdateBusinessInfo](routerGroup, dependencies, apiGenerator)
	registerRoute[[]service.AgentFAQ, feature_wa_business_agent.ListFAQs](routerGroup, dependencies, apiGenerator)
	registerRoute[[]dto_wa.BusinessAgentKeyword, feature_wa_business_agent.ListKeywords](routerGroup, dependencies, apiGenerator)
	registerRoute[*dto_wa.BusinessAgentKeyword, feature_wa_business_agent.CreateKeyword](routerGroup, dependencies, apiGenerator)
	registerRoute[*dto_wa.BusinessAgentKeyword, feature_wa_business_agent.UpdateKeyword](routerGroup, dependencies, apiGenerator)
	registerRoute[any, feature_wa_business_agent.DeleteKeyword](routerGroup, dependencies, apiGenerator)
	registerRoute[*service.AgentFAQ, feature_wa_business_agent.UpdateFAQ](routerGroup, dependencies, apiGenerator)
	registerRoute[any, feature_wa_business_agent.DeleteFAQ](routerGroup, dependencies, apiGenerator)
	registerRoute[*service.AgentFAQ, feature_wa_business_agent.CreateFAQ](routerGroup, dependencies, apiGenerator)
	registerRoute[[]service.AgentFile, feature_wa_business_agent.ListFiles](routerGroup, dependencies, apiGenerator)
	registerCreateAgentFileRoute(routerGroup, dependencies, apiGenerator)
	registerRoute[any, feature_wa_business_agent.DeleteFile](routerGroup, dependencies, apiGenerator)
	registerRoute[[]service.AgentWebsite, feature_wa_business_agent.ListWebsites](routerGroup, dependencies, apiGenerator)
	registerRoute[*service.AgentWebsite, feature_wa_business_agent.UpdateWebsite](routerGroup, dependencies, apiGenerator)
	registerRoute[*service.AgentWebsite, feature_wa_business_agent.CreateWebsite](routerGroup, dependencies, apiGenerator)
	registerRoute[any, feature_wa_business_agent.DeleteWebsite](routerGroup, dependencies, apiGenerator)
	registerRoute[*service.AgentWebsite, feature_wa_business_agent.GetWebsite](routerGroup, dependencies, apiGenerator)
	registerRoute[[]service.AgentSkill, feature_wa_business_agent.ListSkills](routerGroup, dependencies, apiGenerator)
	registerRoute[[]service.AgentConnector, feature_wa_business_agent.ListConnectors](routerGroup, dependencies, apiGenerator)
	registerRoute[*service.AgentConnector, feature_wa_business_agent.CreateConnector](routerGroup, dependencies, apiGenerator)
	registerRoute[*service.AgentConnector, feature_wa_business_agent.UpdateConnector](routerGroup, dependencies, apiGenerator)
	registerRoute[any, feature_wa_business_agent.DeleteConnector](routerGroup, dependencies, apiGenerator)
	registerRoute[*service.AgentConnectorLog, feature_wa_business_agent.GetConnectorLog](routerGroup, dependencies, apiGenerator)
	registerRoute[*service.AgentSkill, feature_wa_business_agent.UpdateSkill](routerGroup, dependencies, apiGenerator)
	registerRoute[*service.AgentSkill, feature_wa_business_agent.CreateSkill](routerGroup, dependencies, apiGenerator)
	registerRoute[any, feature_wa_business_agent.AddCommonSkills](routerGroup, dependencies, apiGenerator)
	registerRoute[any, feature_wa_business_agent.DeleteSkill](routerGroup, dependencies, apiGenerator)
	registerRoute[[]service.AgentUISkill, feature_wa_business_agent.ListUISkills](routerGroup, dependencies, apiGenerator)
	registerRoute[*service.AgentUISkill, feature_wa_business_agent.CreateUISkill](routerGroup, dependencies, apiGenerator)
	registerRoute[*service.AgentTestResponse, feature_wa_business_agent.Test](routerGroup, dependencies, apiGenerator)
	registerRoute[*service.AgentSetting, feature_wa_business_agent.GetSetting](routerGroup, dependencies, apiGenerator)
	registerRoute[*service.AgentSetting, feature_wa_business_agent.UpdateSetting](routerGroup, dependencies, apiGenerator)
	registerRoute[any, feature_wa_business_agent.PassControl](routerGroup, dependencies, apiGenerator)
}

func registerCreateAgentFileRoute(routerGroup *gin.RouterGroup, dependencies *service.Dependencies, apiGenerator *feature.APIGenerator) {
	request := feature_wa_business_agent.CreateFile{}
	settings := request.APISettings()
	_ = apiGenerator.AddEndpoint(request, reflect.TypeFor[*service.AgentFile]())
	routerGroup.POST(settings.Path, func(ctx *gin.Context) {
		startAt := time.Now()
		ctx.Request.Body = http.MaxBytesReader(ctx.Writer, ctx.Request.Body, helper.MaxUploadFileSizeBytes)
		if err := ctx.Request.ParseMultipartForm(1 << 20); err != nil {
			response := dto.NewInvalidInputResponse[*service.AgentFile]([]exception.InputException{{Field: "body", Message: "expected multipart form data with a file part"}})
			finalizeResponse(ctx, dependencies.Logger, &response.ResponseBase, response.Error, startAt)
			ctx.AbortWithStatusJSON(response.StatusCode, response)
			return
		}
		if ctx.Request.MultipartForm != nil {
			defer ctx.Request.MultipartForm.RemoveAll()
		}
		request := feature_wa_business_agent.CreateFile{}
		if err := ctx.ShouldBindUri(&request); err != nil {
			response := dto.NewInvalidInputResponse[*service.AgentFile]([]exception.InputException{{Field: "uri", Message: err.Error()}})
			finalizeResponse(ctx, dependencies.Logger, &response.ResponseBase, response.Error, startAt)
			ctx.AbortWithStatusJSON(response.StatusCode, response)
			return
		}
		fileHeader, err := ctx.FormFile("file")
		if err != nil {
			response := dto.NewInvalidInputResponse[*service.AgentFile]([]exception.InputException{{Field: "file", Message: "file part is required"}})
			finalizeResponse(ctx, dependencies.Logger, &response.ResponseBase, response.Error, startAt)
			ctx.AbortWithStatusJSON(response.StatusCode, response)
			return
		}
		file, err := fileHeader.Open()
		if err != nil {
			response := dto.NewInvalidInputResponse[*service.AgentFile]([]exception.InputException{{Field: "file", Message: "failed to open uploaded file"}})
			finalizeResponse(ctx, dependencies.Logger, &response.ResponseBase, response.Error, startAt)
			ctx.AbortWithStatusJSON(response.StatusCode, response)
			return
		}
		content, err := io.ReadAll(file)
		_ = file.Close()
		if err != nil {
			response := dto.NewInvalidInputResponse[*service.AgentFile]([]exception.InputException{{Field: "file", Message: "failed to read uploaded file"}})
			finalizeResponse(ctx, dependencies.Logger, &response.ResponseBase, response.Error, startAt)
			ctx.AbortWithStatusJSON(response.StatusCode, response)
			return
		}
		request.FileName = ctx.PostForm("file_name")
		if strings.TrimSpace(request.FileName) == "" {
			request.FileName = fileHeader.Filename
		}
		request.Content = content
		var user *dto_account.User
		if userValue, exists := ctx.Get(cfg.Default().Site.HTTPRequestUserKey); exists {
			user = userValue.(*dto_account.User)
		}
		response := request.Handle(ctx.Request.Context(), user, dependencies)
		finalizeResponse(ctx, dependencies.Logger, &response.ResponseBase, response.Error, startAt)
		ctx.JSON(response.StatusCode, response)
	})
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
		startAt := time.Now()
		rawBody, err := ctx.GetRawData()
		if err != nil {
			response := dto.NewFailedResponse[any](http.StatusBadRequest, "failed to read raw body", err)
			finalizeResponse(ctx, dependencies.Logger, &response.ResponseBase, response.Error, startAt)
			ctx.AbortWithStatusJSON(response.StatusCode, response)
			return
		}
		if cfg.Default().Site.Environment != types.EnvironmentDevelop && !helper.VerifyWhatsAppWebhookSignature(ctx.GetHeader("X-Hub-Signature-256"), rawBody, cfg.Default().WhatsApp.AppSecret) {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, dto.NewFailedResponse[any](http.StatusUnauthorized, "invalid WhatsApp webhook signature", nil))
			return
		}
		requestObject := feature_wa_webhook.Receive{RawBody: string(rawBody)}
		responseObject := requestObject.Handle(ctx.Request.Context(), nil, dependencies)
		finalizeResponse(ctx, dependencies.Logger, &responseObject.ResponseBase, responseObject.Error, startAt)
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
		startAt := time.Now()
		ctx.Request.Body = http.MaxBytesReader(ctx.Writer, ctx.Request.Body, helper.MaxUploadFileSizeBytes)
		content, err := ctx.GetRawData()
		if err != nil {
			response := dto.NewFailedResponse[*service.WhatsAppTemplateHeaderSampleUploadResponse](http.StatusBadRequest, "failed to read sample content, it may exceed the 15MB size limit", err)
			finalizeResponse(ctx, dependencies.Logger, &response.ResponseBase, response.Error, startAt)
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
		finalizeResponse(ctx, dependencies.Logger, &response.ResponseBase, response.Error, startAt)
		ctx.JSON(response.StatusCode, response)
	})
}

func registerMediaUploadRoute(routerGroup *gin.RouterGroup, dependencies *service.Dependencies, apiGenerator *feature.APIGenerator) {
	settings := feature_wa_message.UploadMedia{}.APISettings()
	_ = apiGenerator.AddEndpoint(feature_wa_message.UploadMedia{}, reflect.TypeFor[*feature_wa_message.Media]())
	routerGroup.POST(settings.Path, func(ctx *gin.Context) {
		startAt := time.Now()
		ctx.Request.Body = http.MaxBytesReader(ctx.Writer, ctx.Request.Body, helper.MaxUploadFileSizeBytes)
		content, err := ctx.GetRawData()
		if err != nil {
			response := feature_wa_message.UploadMediaRequestError(err)
			finalizeResponse(ctx, dependencies.Logger, &response.ResponseBase, response.Error, startAt)
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
		finalizeResponse(ctx, dependencies.Logger, &response.ResponseBase, response.Error, startAt)
		ctx.JSON(response.StatusCode, response)
	})
}

func finalizeResponse(ctx *gin.Context, logger *service.Logger, response *dto.ResponseBase, err error, startAt time.Time) {
	response.StartAt = startAt
	response.EndAt = time.Now()
	response.TimeTaken = helper.GetTimeDifferenceInMS(response.EndAt, startAt)
	response.RequestId = middleware.GetRequestID(ctx)
	logResponseError(ctx, logger, *response, err)
}
