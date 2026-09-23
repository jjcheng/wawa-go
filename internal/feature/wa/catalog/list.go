package feature_wa_catalog

import (
	"context"
	"net/http"

	"github.com/jjcheng/wawa-go/internal/cfg"
	"github.com/jjcheng/wawa-go/internal/dto"
	dto_account "github.com/jjcheng/wawa-go/internal/dto/account"
	"github.com/jjcheng/wawa-go/internal/exception"
	"github.com/jjcheng/wawa-go/internal/feature"
	"github.com/jjcheng/wawa-go/internal/helper"
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
		return dto.NewFailedResponse[[]service.WhatsAppProductCatalog](http.StatusForbidden, types.ExceptionMessageForbidden, nil)
	}
	if user.Type != types.UserTypeMaster {
		return dto.NewFailedResponse[[]service.WhatsAppProductCatalog](http.StatusUnauthorized, types.ExceptionMessageUnauthorized, nil)
	}
	if user.WA == nil || user.WA.BusinessPortfolio == nil {
		return dto.NewFailedResponse[[]service.WhatsAppProductCatalog](http.StatusUnauthorized, types.ExceptionMessageUnauthorized, nil)
	}
	if inputErrors := list.Validate(); len(inputErrors) > 0 {
		return dto.NewInvalidInputResponse[[]service.WhatsAppProductCatalog](inputErrors)
	}
	catalogs, err := dependencies.Whatsapp.ListCatalogs(ctx, user.WA.BusinessPortfolio.MetaBusinessPortfolioId, user.WA.BusinessPortfolioAccessToken)
	if err != nil {
		return dto.NewFailedResponse[[]service.WhatsAppProductCatalog](http.StatusBadGateway, err.Error(), err)
	}
	// retrive catalogs in our db, not every Meta catalog is in db, only those need to create website
	commerceCatalogs, err := dependencies.UnitOfWork.CommerceCatalogRepository().ListByBusinessAccountId(ctx, user.WA.BusinessAccount.Id)
	if err != nil {
		return dto.NewFailedResponse[[]service.WhatsAppProductCatalog](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
	}
	for _, commerceCatalog := range commerceCatalogs {
		if metaCatalogIndex := helper.IndexOf(catalogs, func(c service.WhatsAppProductCatalog) bool {
			return c.ID == commerceCatalog.MetaId
		}); metaCatalogIndex != nil {
			catalogs[*metaCatalogIndex].WebsiteDomainName = commerceCatalog.WebsiteDomainName
			catalogs[*metaCatalogIndex].WebsiteProductsLastSyncedAt = commerceCatalog.WebsiteProductsLastSyncedAt
			catalogs[*metaCatalogIndex].WebsiteURL = helper.GetWebhsiteFullUrl(commerceCatalog.WebsiteDomainName, cfg.Default().Commerce.WebsiteDomain)
			catalogs[*metaCatalogIndex].WebsiteStatus = string(commerceCatalog.WebsiteStatus)
		}
		if commerceCatalog.Subscribed {
			continue
		}
		err := dependencies.Whatsapp.SubscribeCatalog(ctx, commerceCatalog.MetaId, user.WA.BusinessPortfolioAccessToken)
		if err != nil {
			dependencies.Logger.ErrorFunction(err, commerceCatalog.MetaId)
			continue // no need to return error
		}
		commerceCatalog.Subscribed = true
		if err = dependencies.UnitOfWork.CommerceCatalogRepository().Update(ctx, &commerceCatalog); err != nil {
			dependencies.Logger.ErrorFunction(err, commerceCatalog)
		}
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
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageForbidden, http.StatusForbidden)),
			feature.NewAPIError(*exception.NewCustomException("you are not authorized", http.StatusUnauthorized)),
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageBadGateway, http.StatusBadGateway)),
		},
	)
}
