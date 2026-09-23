package feature_wa_template

import (
	"context"
	"net/http"
	"strings"

	"github.com/jjcheng/wawa-go/internal/dto"
	dto_account "github.com/jjcheng/wawa-go/internal/dto/account"
	dto_wa "github.com/jjcheng/wawa-go/internal/dto/wa"
	"github.com/jjcheng/wawa-go/internal/exception"
	"github.com/jjcheng/wawa-go/internal/feature"
	"github.com/jjcheng/wawa-go/internal/service"
	"github.com/jjcheng/wawa-go/internal/types"
)

type Get struct {
	Id string `uri:"id" val:"required" description:"id of the template from Meta"`
}

func (get *Get) Validate() []exception.InputException {
	get.Id = strings.TrimSpace(get.Id)
	if get.Id == "" {
		return []exception.InputException{exception.NewInputException("id", "missing template ID")}
	}
	return nil
}

func (get Get) Handle(ctx context.Context, user *dto_account.User, dependencies *service.Dependencies) dto.Response[*dto_wa.Template] {
	if user == nil {
		return dto.NewFailedResponse[*dto_wa.Template](http.StatusForbidden, types.ExceptionMessageForbidden, nil)
	}
	if user.WA == nil || user.WA.BusinessAccount == nil || user.WA.BusinessPortfolio == nil || user.WA.BusinessPortfolioAccessToken == "" {
		return dto.NewFailedResponse[*dto_wa.Template](http.StatusUnauthorized, types.ExceptionMessageUnauthorized, nil)
	}
	if inputErrors := get.Validate(); len(inputErrors) > 0 {
		return dto.NewInvalidInputResponse[*dto_wa.Template](inputErrors)
	}
	template, err := dependencies.Whatsapp.GetTemplate(ctx, get.Id, user.WA.BusinessPortfolioAccessToken)
	if err != nil {
		return dto.NewFailedResponse[*dto_wa.Template](http.StatusBadGateway, err.Error(), err)
	}
	template.ByAPI = template.TemplateBase.ByAPI()
	template.MetaEditTemplateUrl = template.GetMetaEditTemplateUrl(user.WA.BusinessPortfolio.MetaBusinessPortfolioId, user.WA.BusinessAccount.WABAId)
	return dto.NewSuccessResponse(template)
}

func (Get) APISettings() feature.APISettings {
	return feature.NewAPISettings(
		"Get WhatsApp template",
		"Gets a WhatsApp message template by its Meta ID.",
		types.HttpRequestTypeUri,
		http.MethodGet,
		"/v1/wa/templates/:id",
		true,
		true,
		types.APITagWA,
		[]feature.APIError{
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageForbidden, http.StatusForbidden)),
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageUnauthorized, http.StatusUnauthorized)),
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageBadGateway, http.StatusBadGateway)),
		},
	)
}
