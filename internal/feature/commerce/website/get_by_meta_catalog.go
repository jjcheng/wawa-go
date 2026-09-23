package feature_commerce_website

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/jjcheng/wawa-go/internal/dto"
	dto_account "github.com/jjcheng/wawa-go/internal/dto/account"
	dto_commerce "github.com/jjcheng/wawa-go/internal/dto/commerce"
	"github.com/jjcheng/wawa-go/internal/exception"
	"github.com/jjcheng/wawa-go/internal/feature"
	"github.com/jjcheng/wawa-go/internal/service"
	"github.com/jjcheng/wawa-go/internal/types"
	"gorm.io/gorm"
)

// 1 catalog can only have 1 website, 1 business account can only have 1 catalog
type GetByMetaCatalogId struct {
	MetaCatalogId string `form:"meta_catalog_id" val:"required" description:"Meta catalog id associated with the website"`
}

func (getByMetaCatalogId *GetByMetaCatalogId) Validate() []exception.InputException {
	getByMetaCatalogId.MetaCatalogId = strings.TrimSpace(getByMetaCatalogId.MetaCatalogId)
	if getByMetaCatalogId.MetaCatalogId == "" {
		return []exception.InputException{exception.NewInputException("meta_catalog_id", "missing Meta catalog id")}
	}
	return nil
}

func (getByMetaCatalogId GetByMetaCatalogId) Handle(ctx context.Context, user *dto_account.User, dependencies *service.Dependencies) dto.Response[*dto_commerce.Website] {
	if user == nil {
		return dto.NewFailedResponse[*dto_commerce.Website](http.StatusForbidden, types.ExceptionMessageForbidden, nil)
	}
	if user.WA == nil || user.WA.BusinessAccount == nil {
		return dto.NewFailedResponse[*dto_commerce.Website](http.StatusUnauthorized, types.ExceptionMessageUnauthorized, nil)
	}
	if inputErrors := getByMetaCatalogId.Validate(); len(inputErrors) > 0 {
		return dto.NewInvalidInputResponse[*dto_commerce.Website](inputErrors)
	}
	website, err := dependencies.UnitOfWork.CommerceWebsiteRepository().GetByMetaCatalogId(ctx, getByMetaCatalogId.MetaCatalogId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return dto.NewFailedResponse[*dto_commerce.Website](http.StatusNotFound, "website not found", nil)
		}
		return dto.NewFailedResponse[*dto_commerce.Website](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
	}
	// check this website belongs to the user
	if website.BusinessAccountId != user.WA.BusinessAccount.Id {
		return dto.NewFailedResponse[*dto_commerce.Website](http.StatusUnauthorized, types.ExceptionMessageUnauthorized, nil)
	}
	result := dto_commerce.NewWebsite(*website)
	return dto.NewSuccessResponse(&result)
}

func (GetByMetaCatalogId) APISettings() feature.APISettings {
	return feature.NewAPISettings(
		"Get commerce website by catalog",
		"Gets the website stored for a Meta catalog.",
		types.HttpRequestTypeQuery,
		http.MethodGet,
		"/v1/commerce/websites/by-meta-catalog-id",
		true,
		true,
		types.APITagCommerce,
		[]feature.APIError{
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageForbidden, http.StatusForbidden)),
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageUnauthorized, http.StatusUnauthorized)),
			feature.NewAPIError(*exception.NewCustomException("website not found", http.StatusNotFound)),
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageInternalServerError, http.StatusInternalServerError)),
		},
	)
}
