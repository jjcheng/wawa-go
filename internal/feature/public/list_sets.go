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

// get sets by website id, since each website has only 1 catalog, it's ok this way
type ListSets struct {
}

func (listSets *ListSets) Validate() []exception.InputException {
	return nil
}

func (listSets ListSets) Handle(ctx context.Context, _ *dto_account.User, dependencies *service.Dependencies) dto.Response[[]dto_commerce.Set] {
	if inputErrors := listSets.Validate(); len(inputErrors) > 0 {
		return dto.NewInvalidInputResponse[[]dto_commerce.Set](inputErrors)
	}
	website := getWebsiteFromContext(ctx)
	if website == nil {
		return dto.NewFailedResponse[[]dto_commerce.Set](http.StatusNotFound, "website not found", nil)
	}
	catalog, err := dependencies.UnitOfWork.CommerceCatalogRepository().GetByWebsiteId(ctx, website.Id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return dto.NewFailedResponse[[]dto_commerce.Set](http.StatusNotFound, "catalog not found", nil)
		}
		return dto.NewFailedResponse[[]dto_commerce.Set](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
	}
	sets, err := dependencies.UnitOfWork.CommerceSetRepository().ListByCatalogId(ctx, catalog.Id)
	if err != nil {
		return dto.NewFailedResponse[[]dto_commerce.Set](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
	}
	items := make([]dto_commerce.Set, 0, len(sets))
	for _, set := range sets {
		items = append(items, dto_commerce.NewProductSet(set))
	}
	return dto.NewSuccessResponse(items)
}

func (ListSets) APISettings() feature.APISettings {
	return feature.NewAPISettings(
		"List product sets in a public website",
		"List all product sets in a public website.",
		types.HttpRequestTypeQuery,
		http.MethodGet,
		"/v1/public/sets",
		false,
		true,
		types.APITagPublic,
		[]feature.APIError{
			feature.NewAPIError(*exception.NewCustomException("catalog not found", http.StatusNotFound)),
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageInternalServerError, http.StatusInternalServerError)),
		},
	)
}
