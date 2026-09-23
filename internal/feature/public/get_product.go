package feature_public

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

type GetProduct struct {
	Id int32 `uri:"id" val:"required" description:"id of the product"`
}

func (getProduct *GetProduct) Validate() []exception.InputException {
	if getProduct.Id <= 0 {
		return []exception.InputException{exception.NewInputException("id", "invalid product id")}
	}
	return nil
}

func (getProduct GetProduct) Handle(ctx context.Context, _ *dto_account.User, dependencies *service.Dependencies) dto.Response[*dto_commerce.GenericProduct] {
	if inputErrors := getProduct.Validate(); len(inputErrors) > 0 {
		return dto.NewInvalidInputResponse[*dto_commerce.GenericProduct](inputErrors)
	}
	website := getWebsiteFromContext(ctx)
	if website == nil {
		return dto.NewFailedResponse[*dto_commerce.GenericProduct](http.StatusNotFound, "website not found", nil)
	}
	// get catalog
	catalog, err := dependencies.UnitOfWork.CommerceCatalogRepository().GetByWebsiteId(ctx, website.Id)
	if err != nil {
		return dto.NewFailedResponse[*dto_commerce.GenericProduct](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
	}
	// get product
	product, err := dependencies.UnitOfWork.CommerceGenericProductRepository().GetById(ctx, getProduct.Id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return dto.NewFailedResponse[*dto_commerce.GenericProduct](http.StatusNotFound, "product not found", nil)
		}
		dependencies.Logger.ErrorFunction(err, getProduct.Id)
		return dto.NewFailedResponse[*dto_commerce.GenericProduct](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
	}
	// get set
	set, err := dependencies.UnitOfWork.CommerceSetRepository().GetById(ctx, product.SetId)
	if err != nil {
		return dto.NewFailedResponse[*dto_commerce.GenericProduct](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
	}
	if set.CatalogId != catalog.Id {
		return dto.NewFailedResponse[*dto_commerce.GenericProduct](http.StatusNotFound, "product not found", nil)
	}
	result := dto_commerce.NewProduct(*product)
	return dto.NewSuccessResponse(&result)
}

func (GetProduct) APISettings() feature.APISettings {
	return feature.NewAPISettings(
		"Get public product",
		"Gets a product from a commerce product set.",
		types.HttpRequestTypeUri,
		http.MethodGet,
		"/v1/public/products/:id",
		false,
		true,
		types.APITagPublic,
		[]feature.APIError{
			feature.NewAPIError(*exception.NewCustomException("product not found", http.StatusNotFound)),
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageInternalServerError, http.StatusInternalServerError)),
		},
	)
}
