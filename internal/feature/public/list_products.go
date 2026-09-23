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

type ListProducts struct {
	SetId    int32 `form:"set_id" val:"required" description:"id of the set"`
	Page     int32 `form:"page" description:"page from 1"`
	PageSize int32 `form:"page_size" description:"page size default 10"`
}

func (listProducts *ListProducts) Validate() []exception.InputException {
	if listProducts.Page <= 0 {
		listProducts.Page = 1
	}
	if listProducts.PageSize <= 0 || listProducts.Page > 20 {
		listProducts.PageSize = 10
	}
	if listProducts.SetId <= 0 {
		return []exception.InputException{exception.NewInputException("set_id", "invalid set id")}
	}
	return nil
}

func (listProducts ListProducts) Handle(ctx context.Context, _ *dto_account.User, dependencies *service.Dependencies) dto.Response[*dto.ListResponse[dto_commerce.GenericProduct]] {
	if inputErrors := listProducts.Validate(); len(inputErrors) > 0 {
		return dto.NewInvalidInputResponse[*dto.ListResponse[dto_commerce.GenericProduct]](inputErrors)
	}
	website := getWebsiteFromContext(ctx)
	if website == nil {
		return dto.NewFailedResponse[*dto.ListResponse[dto_commerce.GenericProduct]](http.StatusNotFound, "website not found", nil)
	}
	set, err := dependencies.UnitOfWork.CommerceSetRepository().GetById(ctx, listProducts.SetId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return dto.NewFailedResponse[*dto.ListResponse[dto_commerce.GenericProduct]](http.StatusNotFound, "set not found", nil)
		}
		return dto.NewFailedResponse[*dto.ListResponse[dto_commerce.GenericProduct]](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
	}
	// get catalog
	catalog, err := dependencies.UnitOfWork.CommerceCatalogRepository().GetByWebsiteId(ctx, website.Id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return dto.NewFailedResponse[*dto.ListResponse[dto_commerce.GenericProduct]](http.StatusNotFound, "catalog not found", nil)
		}
		return dto.NewFailedResponse[*dto.ListResponse[dto_commerce.GenericProduct]](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
	}
	if set.CatalogId != catalog.Id {
		return dto.NewFailedResponse[*dto.ListResponse[dto_commerce.GenericProduct]](http.StatusNotFound, "set not found", nil)
	}
	products, totalItems, totalPages, err := dependencies.UnitOfWork.CommerceGenericProductRepository().List(ctx, set.CatalogId, listProducts.SetId, "", "id", true, int(listProducts.Page), int(listProducts.PageSize))
	if err != nil {
		return dto.NewFailedResponse[*dto.ListResponse[dto_commerce.GenericProduct]](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
	}
	items := make([]dto_commerce.GenericProduct, 0, len(products))
	for _, product := range products {
		items = append(items, dto_commerce.NewProduct(product))
	}
	response := dto.NewPagedListResponse(items, totalPages, totalItems)
	return dto.NewSuccessResponse(&response)
}

func (ListProducts) APISettings() feature.APISettings {
	return feature.NewAPISettings(
		"List public website products",
		"Lists products in a generic product set.",
		types.HttpRequestTypeQuery,
		http.MethodGet,
		"/v1/public/generic-products",
		false,
		true,
		types.APITagPublic,
		[]feature.APIError{
			feature.NewAPIError(*exception.NewCustomException("set not found", http.StatusNotFound)),
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageInternalServerError, http.StatusInternalServerError)),
		},
	)
}
