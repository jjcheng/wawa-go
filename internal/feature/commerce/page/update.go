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
	"github.com/jjcheng/wawa-go/internal/helper"
	"github.com/jjcheng/wawa-go/internal/service"
	"github.com/jjcheng/wawa-go/internal/types"
	"gorm.io/gorm"
)

type Update struct {
	Id int32 `uri:"id" val:"required" description:"id of the page"`
	Create
}

func (update *Update) Validate() []exception.InputException {
	inputErrors := update.Create.Validate()
	if update.Id <= 0 {
		inputErrors = append(inputErrors, exception.NewInputException("id", "missing id"))
	}
	return inputErrors
}

func (update Update) Handle(ctx context.Context, user *dto_account.User, dependencies *service.Dependencies) dto.Response[*dto_commerce.Page] {
	if user == nil {
		return dto.NewFailedResponse[*dto_commerce.Page](http.StatusForbidden, types.ExceptionMessageForbidden, nil)
	}
	if user.Type != types.UserTypeMaster || user.WA == nil || user.WA.BusinessAccount == nil {
		return dto.NewFailedResponse[*dto_commerce.Page](http.StatusUnauthorized, types.ExceptionMessageUnauthorized, nil)
	}
	if inputErrors := update.Validate(); len(inputErrors) > 0 {
		return dto.NewInvalidInputResponse[*dto_commerce.Page](inputErrors)
	}
	page, err := dependencies.UnitOfWork.CommercePageRepository().GetById(ctx, update.Id)
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
	if website.BusinessAccountId != user.WA.BusinessAccount.Id {
		return dto.NewFailedResponse[*dto_commerce.Page](http.StatusUnauthorized, types.ExceptionMessageUnauthorized, nil)
	}
	page.Slug = update.Slug
	page.Title = update.Title
	page.Description = update.Description
	page.Content = update.Content
	page.Nav = update.Nav
	// add image urls
	page.ImageUrls = append(page.ImageUrls, update.ImageUrls...)
	page.ImageUrls = helper.Distinct(page.ImageUrls)
	if err := dependencies.UnitOfWork.CommercePageRepository().Update(ctx, page); err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return dto.NewFailedResponse[*dto_commerce.Page](http.StatusConflict, "page slug is already in use", nil)
		}
		return dto.NewFailedResponse[*dto_commerce.Page](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
	}
	result := dto_commerce.NewPage(*page)
	return dto.NewSuccessResponse(&result)
}

func (Update) APISettings() feature.APISettings {
	return feature.NewAPISettings(
		"Update commerce website page",
		"Updates a page belonging to a website in the authenticated user's business account.",
		types.HttpRequestTypeUriJSON,
		http.MethodPatch,
		"/v1/commerce/pages/:id",
		true,
		true,
		types.APITagCommerce,
		[]feature.APIError{
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageForbidden, http.StatusForbidden)),
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageUnauthorized, http.StatusUnauthorized)),
			feature.NewAPIError(*exception.NewCustomException("page not found", http.StatusNotFound)),
			feature.NewAPIError(*exception.NewCustomException("page slug is already in use", http.StatusConflict)),
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageInternalServerError, http.StatusInternalServerError)),
		},
	)
}
