package feature_commerce_page

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
	Id int32 `uri:"id" val:"required" description:"id of the page"`
}

func (get *Get) Validate() []exception.InputException {
	if get.Id <= 0 {
		return []exception.InputException{exception.NewInputException("id", "invalid page id")}
	}
	return nil
}

func (get Get) Handle(ctx context.Context, user *dto_account.User, dependencies *service.Dependencies) dto.Response[*dto_commerce.Page] {
	if user == nil {
		return dto.NewFailedResponse[*dto_commerce.Page](http.StatusForbidden, types.ExceptionMessageForbidden, nil)
	}
	if user.WA == nil {
		return dto.NewFailedResponse[*dto_commerce.Page](http.StatusUnauthorized, types.ExceptionMessageUnauthorized, nil)
	}
	if inputErrors := get.Validate(); len(inputErrors) > 0 {
		return dto.NewInvalidInputResponse[*dto_commerce.Page](inputErrors)
	}
	page, err := dependencies.UnitOfWork.CommercePageRepository().GetById(ctx, get.Id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return dto.NewFailedResponse[*dto_commerce.Page](http.StatusNotFound, "page not found", nil)
		}
		return dto.NewFailedResponse[*dto_commerce.Page](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
	}
	website, err := dependencies.UnitOfWork.CommerceWebsiteRepository().GetById(ctx, page.WebsiteId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return dto.NewFailedResponse[*dto_commerce.Page](http.StatusNotFound, "page not found", nil)
		}
		return dto.NewFailedResponse[*dto_commerce.Page](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
	}
	if website.BusinessAccountId != user.BusinessAccountId {
		return dto.NewFailedResponse[*dto_commerce.Page](http.StatusNotFound, "page not found", nil)
	}
	result := dto_commerce.NewPage(*page)
	return dto.NewSuccessResponse(&result)
}

func (Get) APISettings() feature.APISettings {
	return feature.NewAPISettings(
		"Get a website page",
		"Get a page belonging to a website in the business account.",
		types.HttpRequestTypeUri,
		http.MethodGet,
		"/v1/commerce/pages/:id",
		true,
		true,
		types.APITagCommerce,
		[]feature.APIError{
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageForbidden, http.StatusForbidden)),
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageUnauthorized, http.StatusUnauthorized)),
			feature.NewAPIError(*exception.NewCustomException("page not found", http.StatusNotFound)),
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageInternalServerError, http.StatusInternalServerError)),
		},
	)
}
