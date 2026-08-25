package controller

import (
	"net/http"

	"github.com/jjcheng/wawa-go/internal/cfg"
	dto_wa "github.com/jjcheng/wawa-go/internal/dto/wa"
	"github.com/jjcheng/wawa-go/internal/feature"
	feature_wa "github.com/jjcheng/wawa-go/internal/feature/wa"
	feature_wa_business_account "github.com/jjcheng/wawa-go/internal/feature/wa/business_account"
	feature_wa_business_portfolio "github.com/jjcheng/wawa-go/internal/feature/wa/business_portfolio"
	feature_wa_user_phone_number "github.com/jjcheng/wawa-go/internal/feature/wa/user_phone_number"
	"github.com/jjcheng/wawa-go/internal/middleware"
	"github.com/jjcheng/wawa-go/internal/service"

	"github.com/gin-gonic/gin"
)

func registerWAController(authGroup, unauthGroup *gin.RouterGroup, dependencies *service.Dependencies, apiGenerator *feature.APIGenerator) {
	// verify endpoint
	unauthGroup.GET(feature_wa.Verify{}.APISettings().Path, middleware.BindRequest[string, feature_wa.Verify](), func(ctx *gin.Context) {
		requestObject := ctx.MustGet(cfg.Default().Site.HTTPRequestItemKey).(feature_wa.Verify)
		responseObject := requestObject.Handle(ctx.Request.Context(), nil, dependencies)
		if !responseObject.Success {
			ctx.AbortWithStatusJSON(responseObject.StatusCode, responseObject)
			return
		}
		ctx.String(responseObject.StatusCode, responseObject.Data)
	})
	// receive something, can be messages, status etc...
	unauthGroup.POST(feature_wa.Receive{}.APISettings().Path, func(ctx *gin.Context) {
		rawBody, err := ctx.GetRawData()
		if err != nil {
			ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"message": "failed to read raw body"})
			return
		}
		requestObject := feature_wa.Receive{RawBody: string(rawBody)}
		responseObject := requestObject.Handle(ctx.Request.Context(), nil, dependencies)
		if !responseObject.Success {
			ctx.AbortWithStatusJSON(responseObject.StatusCode, responseObject)
			return
		}
		ctx.JSON(responseObject.StatusCode, responseObject)
	})
	// embedded signup
	registerRoute[*feature_wa.EmbeddedSignupResponse, feature_wa.EmbeddedSignup](unauthGroup, dependencies, apiGenerator)
	// business portfolio
	registerRoute[*dto_wa.BusinessPortfolio, feature_wa_business_portfolio.Get](authGroup, dependencies, apiGenerator)
	registerRoute[*dto_wa.BusinessPortfolio, feature_wa_business_portfolio.UpdateName](authGroup, dependencies, apiGenerator)
	// business account
	registerRoute[*dto_wa.BusinessAccount, feature_wa_business_account.Get](authGroup, dependencies, apiGenerator)
	registerRoute[*dto_wa.BusinessAccount, feature_wa_business_account.UpdateName](authGroup, dependencies, apiGenerator)
	// phone number
	registerRoute[*dto_wa.UserPhoneNumber, feature_wa_user_phone_number.Create](authGroup, dependencies, apiGenerator)
}
