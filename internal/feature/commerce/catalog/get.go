package feature_commerce_catalog

import (
	"context"
	"errors"
	"net/http"

	"github.com/jjcheng/wawa-go/internal/dto"
	dto_account "github.com/jjcheng/wawa-go/internal/dto/account"
	dto_commerce "github.com/jjcheng/wawa-go/internal/dto/commerce"
	"github.com/jjcheng/wawa-go/internal/exception"
	"github.com/jjcheng/wawa-go/internal/feature"
	"github.com/jjcheng/wawa-go/internal/service"
	"github.com/jjcheng/wawa-go/internal/types"
	"gorm.io/gorm"
)

type Get struct {
	Id int32 `uri:"id" val:"required" description:"id of the catalog"`
}

func (get *Get) Validate() []exception.InputException {
	if get.Id <= 0 {
		return []exception.InputException{exception.NewInputException("id", "invalid catalog id")}
	}
	return nil
}

func (get Get) Handle(ctx context.Context, user *dto_account.User, dependencies *service.Dependencies) dto.Response[*dto_commerce.Catalog] {
	if user == nil {
		return dto.NewFailedResponse[*dto_commerce.Catalog](http.StatusForbidden, types.ExceptionMessageForbidden, nil)
	}
	if inputErrors := get.Validate(); len(inputErrors) > 0 {
		return dto.NewInvalidInputResponse[*dto_commerce.Catalog](inputErrors)
	}
	catalog, err := dependencies.UnitOfWork.CommerceCatalogRepository().GetById(ctx, get.Id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return dto.NewFailedResponse[*dto_commerce.Catalog](http.StatusNotFound, "catalog not found", nil)
		}
		return dto.NewFailedResponse[*dto_commerce.Catalog](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
	}
	website, err := dependencies.UnitOfWork.CommerceWebsiteRepository().GetById(ctx, catalog.WebsiteId)
	if err != nil {
		return dto.NewFailedResponse[*dto_commerce.Catalog](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
	}
	if website.BusinessAccountId != user.BusinessAccountId {
		return dto.NewFailedResponse[*dto_commerce.Catalog](http.StatusUnauthorized, types.ExceptionMessageUnauthorized, nil)
	}
	result := dto_commerce.NewCatalog(*catalog)
	return dto.NewSuccessResponse(&result)
}

func (Get) APISettings() feature.APISettings {
	return feature.NewAPISettings(
		"Get a catalog",
		"Get a catalog belonging to the business account.",
		types.HttpRequestTypeUri,
		http.MethodGet,
		"/v1/commerce/catalogs/:id",
		true,
		true,
		types.APITagCommerce,
		[]feature.APIError{
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageForbidden, http.StatusForbidden)),
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageUnauthorized, http.StatusUnauthorized)),
			feature.NewAPIError(*exception.NewCustomException("catalog not found", http.StatusNotFound)),
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageInternalServerError, http.StatusInternalServerError)),
		},
	)
}
