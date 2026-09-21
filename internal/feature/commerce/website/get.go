package feature_commerce_website

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
	Id int32 `uri:"id" val:"required" description:"id of the website"`
}

func (get *Get) Validate() []exception.InputException {
	if get.Id <= 0 {
		return []exception.InputException{exception.NewInputException("id", "invalid website id")}
	}
	return nil
}

func (get Get) Handle(ctx context.Context, user *dto_account.User, dependencies *service.Dependencies) dto.Response[*dto_commerce.Website] {
	if user == nil {
		return dto.NewFailedResponse[*dto_commerce.Website](http.StatusForbidden, "you are not authenticated")
	}
	if user.WA == nil || user.WA.BusinessAccount == nil {
		return dto.NewFailedResponse[*dto_commerce.Website](http.StatusUnauthorized, "you are not authorized")
	}
	if inputErrors := get.Validate(); len(inputErrors) > 0 {
		return dto.NewInvalidInputResponse[*dto_commerce.Website](inputErrors)
	}
	website, err := dependencies.UnitOfWork.CommerceWebsiteRepository().GetById(ctx, get.Id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return dto.NewFailedResponse[*dto_commerce.Website](http.StatusNotFound, "website not found")
		}
		return dto.NewFailedResponse[*dto_commerce.Website](http.StatusInternalServerError, types.ExceptionMessageInternalServerError)
	}
	catalog, err := dependencies.UnitOfWork.CommerceCatalogRepository().GetByWebsiteId(ctx, website.Id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return dto.NewFailedResponse[*dto_commerce.Website](http.StatusNotFound, "catalog not found")
		}
		dependencies.Logger.ErrorFunction(err, website.Id)
		return dto.NewFailedResponse[*dto_commerce.Website](http.StatusInternalServerError, types.ExceptionMessageInternalServerError)
	}
	website.CatalogName = catalog.Name
	website.MetaCatalogId = catalog.MetaId
	d := dto_commerce.NewWebsite(*website)
	return dto.NewSuccessResponse(&d)
}

func (Get) APISettings() feature.APISettings {
	return feature.NewAPISettings(
		"Get commerce website",
		"Gets a website belonging to the authenticated user's business account.",
		types.HttpRequestTypeUri,
		http.MethodGet,
		"/v1/commerce/websites/:id",
		true,
		true,
		types.APITagCommerce,
		[]feature.APIError{
			feature.NewAPIError(*exception.NewCustomException("you are not authenticated", http.StatusForbidden)),
			feature.NewAPIError(*exception.NewCustomException("you are not authorized", http.StatusUnauthorized)),
			feature.NewAPIError(*exception.NewCustomException("website not found", http.StatusNotFound)),
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageInternalServerError, http.StatusInternalServerError)),
		},
	)
}
