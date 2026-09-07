package controller

import (
	"net/http"
	"reflect"

	"github.com/gorilla/websocket"
	"github.com/jjcheng/wawa-go/internal/cfg"
	"github.com/jjcheng/wawa-go/internal/dto"
	dto_account "github.com/jjcheng/wawa-go/internal/dto/account"
	dto_wa "github.com/jjcheng/wawa-go/internal/dto/wa"
	"github.com/jjcheng/wawa-go/internal/feature"
	feature_wa_account "github.com/jjcheng/wawa-go/internal/feature/wa/account"
	feature_wa_business_account "github.com/jjcheng/wawa-go/internal/feature/wa/business_account"
	feature_wa_business_portfolio "github.com/jjcheng/wawa-go/internal/feature/wa/business_portfolio"
	feature_wa_campaign "github.com/jjcheng/wawa-go/internal/feature/wa/campaign"
	feature_wa_message "github.com/jjcheng/wawa-go/internal/feature/wa/message"
	feature_wa_phone_number "github.com/jjcheng/wawa-go/internal/feature/wa/phone_number"
	feature_wa_sample_template "github.com/jjcheng/wawa-go/internal/feature/wa/sample_template"
	feature_wa_template "github.com/jjcheng/wawa-go/internal/feature/wa/template"
	feature_wa_user_phone_number "github.com/jjcheng/wawa-go/internal/feature/wa/user_phone_number"
	feature_wa_webhook "github.com/jjcheng/wawa-go/internal/feature/wa/webhook"
	"github.com/jjcheng/wawa-go/internal/middleware"
	"github.com/jjcheng/wawa-go/internal/service"

	"github.com/gin-gonic/gin"
)

var waMessageStreamUpgrader = websocket.Upgrader{
	CheckOrigin: func(_ *http.Request) bool { return true },
}

func registerWAController(routerGroup *gin.RouterGroup, dependencies *service.Dependencies, apiGenerator *feature.APIGenerator) {
	registerWAMessageStreamRoute(routerGroup, dependencies, apiGenerator)
	// verify endpoint
	routerGroup.GET(feature_wa_webhook.Verify{}.APISettings().Path, middleware.BindRequest[string, feature_wa_webhook.Verify](), func(ctx *gin.Context) {
		requestObject := ctx.MustGet(cfg.Default().Site.HTTPRequestItemKey).(feature_wa_webhook.Verify)
		responseObject := requestObject.Handle(ctx.Request.Context(), nil, dependencies)
		if !responseObject.Success {
			ctx.AbortWithStatusJSON(responseObject.StatusCode, responseObject)
			return
		}
		ctx.String(responseObject.StatusCode, responseObject.Data)
	})
	// receive something, can be messages, status etc...
	routerGroup.POST(feature_wa_webhook.Receive{}.APISettings().Path, func(ctx *gin.Context) {
		rawBody, err := ctx.GetRawData()
		if err != nil {
			ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"message": "failed to read raw body"})
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
	// business portfolio
	registerRoute[*dto_wa.BusinessPortfolio, feature_wa_business_portfolio.Get](routerGroup, dependencies, apiGenerator)
	registerRoute[*dto_wa.BusinessPortfolio, feature_wa_business_portfolio.Update](routerGroup, dependencies, apiGenerator)
	// business account
	registerRoute[*dto_wa.BusinessAccount, feature_wa_business_account.Get](routerGroup, dependencies, apiGenerator)
	registerRoute[*dto_wa.BusinessAccount, feature_wa_business_account.Update](routerGroup, dependencies, apiGenerator)
	// phone number
	registerRoute[[]dto_wa.PhoneNumber, feature_wa_user_phone_number.List](routerGroup, dependencies, apiGenerator)
	registerRoute[*dto_wa.UserPhoneNumber, feature_wa_user_phone_number.Create](routerGroup, dependencies, apiGenerator)
	registerRoute[any, feature_wa_phone_number.Delete](routerGroup, dependencies, apiGenerator)
	// message
	registerRoute[*service.WhatsAppMessageResponse, feature_wa_message.Create](routerGroup, dependencies, apiGenerator)
	registerRoute[*dto.ListResponse[dto_wa.Message], feature_wa_message.List](routerGroup, dependencies, apiGenerator)
	registerRoute[*feature_wa_message.Media, feature_wa_message.GetMedia](routerGroup, dependencies, apiGenerator)
	// template
	registerRoute[*dto.ListResponse[dto_wa.Template], feature_wa_template.List](routerGroup, dependencies, apiGenerator)
	registerRoute[*dto_wa.Template, feature_wa_template.Create](routerGroup, dependencies, apiGenerator)
	registerRoute[*dto_wa.Template, feature_wa_template.CreateFromSample](routerGroup, dependencies, apiGenerator)
	registerRoute[any, feature_wa_template.Delete](routerGroup, dependencies, apiGenerator)
	// sample templates
	registerRoute[[]dto_wa.SampleTemplate, feature_wa_sample_template.List](routerGroup, dependencies, apiGenerator)
	registerRoute[*dto_wa.SampleTemplate, feature_wa_sample_template.Create](routerGroup, dependencies, apiGenerator)
	// analytics
	registerRoute[*dto_wa.MessageAnalytics, feature_wa_business_account.GetUsage](routerGroup, dependencies, apiGenerator)
	registerRoute[[]dto_wa.PhoneNumberMessageAnalytics, feature_wa_phone_number.GetUsage](routerGroup, dependencies, apiGenerator)
	registerRoute[[]dto_wa.TemplateAnalytics, feature_wa_template.GetUsage](routerGroup, dependencies, apiGenerator)
	// campaigns
	registerRoute[*dto.ListResponse[dto_wa.Campaign], feature_wa_campaign.List](routerGroup, dependencies, apiGenerator)
	registerRoute[*dto_wa.Campaign, feature_wa_campaign.Get](routerGroup, dependencies, apiGenerator)
	registerRoute[*dto_wa.Campaign, feature_wa_campaign.Create](routerGroup, dependencies, apiGenerator)
	registerRoute[any, feature_wa_campaign.Archive](routerGroup, dependencies, apiGenerator)
	registerRoute[any, feature_wa_campaign.Cancel](routerGroup, dependencies, apiGenerator)
}

func registerWAMessageStreamRoute(routerGroup *gin.RouterGroup, dependencies *service.Dependencies, apiGenerator *feature.APIGenerator) {
	_ = apiGenerator.AddEndpoint(feature_wa_message.Stream{}, reflect.TypeFor[service.WAMessageStreamEvent]())
	routerGroup.GET("/v1/wa/messages/stream", func(ctx *gin.Context) {
		request := feature_wa_message.Stream{
			PhoneNumberID:       ctx.Query("phone_number_id"),
			CustomerPhoneNumber: ctx.Query("customer_phone_number"),
		}
		var user *dto_account.User
		if userValue, exists := ctx.Get(cfg.Default().Site.HTTPRequestUserKey); exists {
			user = userValue.(*dto_account.User)
		}
		response := request.Handle(ctx.Request.Context(), user, dependencies)
		if !response.Success {
			ctx.AbortWithStatusJSON(response.StatusCode, response)
			return
		}
		connection, err := waMessageStreamUpgrader.Upgrade(ctx.Writer, ctx.Request, nil)
		if err != nil {
			return
		}
		defer connection.Close()
		defer response.Data.Unsubscribe()
		done := make(chan struct{})
		go func() {
			defer close(done)
			for {
				if _, _, err := connection.ReadMessage(); err != nil {
					return
				}
			}
		}()
		for {
			select {
			case message := <-response.Data.Events:
				if err := connection.WriteJSON(message); err != nil {
					return
				}
			case <-ctx.Request.Context().Done():
				return
			case <-done:
				return
			}
		}
	})
}
