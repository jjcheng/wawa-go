package feature_wa_catalog

import (
	"context"
	"net/http"
	"strings"

	"github.com/jjcheng/wawa-go/internal/dto"
	dto_account "github.com/jjcheng/wawa-go/internal/dto/account"
	"github.com/jjcheng/wawa-go/internal/exception"
	"github.com/jjcheng/wawa-go/internal/feature"
	"github.com/jjcheng/wawa-go/internal/service"
	"github.com/jjcheng/wawa-go/internal/types"
)

type Get struct {
	Id string `uri:"id" val:"required" description:"id of the catalog"`
}

func (get *Get) Validate() []exception.InputException {
	get.Id = strings.TrimSpace(get.Id)
	if get.Id == "" {
		return []exception.InputException{exception.NewInputException("id", "missing catalog ID")}
	}
	return nil
}

func (get Get) Handle(ctx context.Context, user *dto_account.User, dependencies *service.Dependencies) dto.Response[*service.WhatsAppProductCatalog] {
	if user == nil {
		return dto.NewFailedResponse[*service.WhatsAppProductCatalog](http.StatusForbidden, types.ExceptionMessageForbidden, nil)
	}
	if user.Type != types.UserTypeMaster {
		return dto.NewFailedResponse[*service.WhatsAppProductCatalog](http.StatusUnauthorized, types.ExceptionMessageUnauthorized, nil)
	}
	if user.WA == nil || user.WA.BusinessPortfolio == nil {
		return dto.NewFailedResponse[*service.WhatsAppProductCatalog](http.StatusUnauthorized, types.ExceptionMessageUnauthorized, nil)
	}
	if inputErrors := get.Validate(); len(inputErrors) > 0 {
		return dto.NewInvalidInputResponse[*service.WhatsAppProductCatalog](inputErrors)
	}
	catalog, err := dependencies.Whatsapp.GetCatalog(ctx, get.Id, user.WA.BusinessPortfolioAccessToken)
	if err != nil {
		return dto.NewFailedResponse[*service.WhatsAppProductCatalog](http.StatusBadGateway, err.Error(), err)
	}
	return dto.NewSuccessResponse(catalog)
}

func (Get) APISettings() feature.APISettings {
	return feature.NewAPISettings(
		"Get WhatsApp product catalog",
		"Get a Meta commerce catalog in the business portfolio by Meta API.",
		types.HttpRequestTypeUri,
		http.MethodGet,
		"/v1/wa/catalogs/:id",
		true,
		true,
		types.APITagWA,
		[]feature.APIError{
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageForbidden, http.StatusForbidden)),
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageUnauthorized, http.StatusUnauthorized)),
			feature.NewAPIError(*exception.NewCustomException("catalog not found", http.StatusNotFound)),
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageBadGateway, http.StatusBadGateway)),
		},
	)
}
