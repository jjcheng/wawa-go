package feature_wa_catalog

import (
	"context"
	"net/http"

	"github.com/jjcheng/wawa-go/internal/dto"
	dto_account "github.com/jjcheng/wawa-go/internal/dto/account"
	"github.com/jjcheng/wawa-go/internal/exception"
	"github.com/jjcheng/wawa-go/internal/feature"
	"github.com/jjcheng/wawa-go/internal/service"
	"github.com/jjcheng/wawa-go/internal/types"
)

type List struct {
}

func (list *List) Validate() []exception.InputException {
	return nil
}

func (list List) Handle(ctx context.Context, user *dto_account.User, dependencies *service.Dependencies) dto.Response[[]service.WhatsAppProductCatalog] {
	if user == nil {
		return dto.NewFailedResponse[[]service.WhatsAppProductCatalog](http.StatusForbidden, "you are not authenticated")
	}
	if user.Type != types.UserTypeMaster {
		return dto.NewFailedResponse[[]service.WhatsAppProductCatalog](http.StatusUnauthorized, "you are not authorized")
	}
	if inputErrors := list.Validate(); len(inputErrors) > 0 {
		return dto.NewInvalidInputResponse[[]service.WhatsAppProductCatalog](inputErrors)
	}
	if user.WA == nil || user.WA.BusinessPortfolio == nil {
		return dto.NewFailedResponse[[]service.WhatsAppProductCatalog](http.StatusUnauthorized, "you are not authorized")
	}
	catalogs, err := dependencies.Whatsapp.ListCatalogs(ctx, user.WA.BusinessPortfolio.MetaBusinessPortfolioId, user.WA.BusinessPortfolioAccessToken)
	if err != nil {
		return dto.NewFailedResponse[[]service.WhatsAppProductCatalog](http.StatusBadGateway, err.Error())
	}
	return dto.NewSuccessResponse(catalogs)
}

func (List) APISettings() feature.APISettings {
	return feature.NewAPISettings(
		"List WhatsApp product catalogs",
		"Lists product catalogs owned by the authenticated user's Meta business portfolio.",
		types.HttpRequestTypeNone,
		http.MethodGet,
		"/v1/wa/catalogs",
		true,
		true,
		types.APITagWA,
		[]feature.APIError{
			feature.NewAPIError(*exception.NewCustomException("you are not authenticated", http.StatusForbidden)),
			feature.NewAPIError(*exception.NewCustomException("you are not authorized", http.StatusUnauthorized)),
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageBadGateway, http.StatusBadGateway)),
		},
	)
}
