package feature_commerce_page

import (
	"context"
	"errors"
	"net/http"
	"regexp"
	"strings"

	dao_commerce "github.com/jjcheng/wawa-go/internal/dao/commerce"
	"github.com/jjcheng/wawa-go/internal/dto"
	dto_account "github.com/jjcheng/wawa-go/internal/dto/account"
	dto_commerce "github.com/jjcheng/wawa-go/internal/dto/commerce"
	"github.com/jjcheng/wawa-go/internal/exception"
	"github.com/jjcheng/wawa-go/internal/feature"
	"github.com/jjcheng/wawa-go/internal/service"
	"github.com/jjcheng/wawa-go/internal/types"
	"gorm.io/gorm"
)

type Create struct {
	WebsiteId   int32    `uri:"id" val:"required" description:"id of the website"`
	Slug        string   `json:"slug" val:"required" description:"slug of the page"`
	Title       string   `json:"title" val:"required" description:"title of the page"`
	Description string   `json:"description" description:"optional description of the page"`
	Content     string   `json:"content" val:"required" description:"content of the page"`
	Nav         bool     `json:"nav" val:"required" description:"show on nav bar or not"`
	ImageUrls   []string `json:"image_urls" description:"uploaded image urls"`
}

func (create *Create) Validate() []exception.InputException {
	create.Slug = strings.ToLower(strings.TrimSpace(create.Slug))
	create.Title = strings.TrimSpace(create.Title)
	create.Description = strings.TrimSpace(create.Description)
	create.Content = strings.TrimSpace(create.Content)
	inputErrors := []exception.InputException{}
	if create.WebsiteId <= 0 {
		inputErrors = append(inputErrors, exception.NewInputException("id", "invalid website id"))
	}
	if create.Slug == "" {
		inputErrors = append(inputErrors, exception.NewInputException("slug", "missing page slug"))
	} else if len(create.Slug) > 120 || !regexp.MustCompile(`^[a-z0-9](?:[a-z0-9-]*[a-z0-9])?$`).MatchString(create.Slug) {
		inputErrors = append(inputErrors, exception.NewInputException("slug", "invalid page slug"))
	}
	if create.Title == "" {
		inputErrors = append(inputErrors, exception.NewInputException("title", "missing page title"))
	}
	if create.Content == "" {
		inputErrors = append(inputErrors, exception.NewInputException("content", "missing page content"))
	}
	return inputErrors
}

func (create Create) Handle(ctx context.Context, user *dto_account.User, dependencies *service.Dependencies) dto.Response[*dto_commerce.Page] {
	if user == nil {
		return dto.NewFailedResponse[*dto_commerce.Page](http.StatusForbidden, types.ExceptionMessageForbidden, nil)
	}
	if user.Type != types.UserTypeMaster {
		return dto.NewFailedResponse[*dto_commerce.Page](http.StatusUnauthorized, types.ExceptionMessageUnauthorized, nil)
	}
	if inputErrors := create.Validate(); len(inputErrors) > 0 {
		return dto.NewInvalidInputResponse[*dto_commerce.Page](inputErrors)
	}
	website, err := dependencies.UnitOfWork.CommerceWebsiteRepository().GetById(ctx, create.WebsiteId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return dto.NewFailedResponse[*dto_commerce.Page](http.StatusNotFound, "website not found", nil)
		}
		return dto.NewFailedResponse[*dto_commerce.Page](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
	}
	if website.BusinessAccountId != user.BusinessAccountId {
		return dto.NewFailedResponse[*dto_commerce.Page](http.StatusUnauthorized, types.ExceptionMessageUnauthorized, nil)
	}
	pagesCount, err := dependencies.UnitOfWork.CommercePageRepository().GetPagesCountByWebsiteId(ctx, website.Id)
	if err != nil {
		return dto.NewFailedResponse[*dto_commerce.Page](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
	}
	page := dao_commerce.Page{
		WebsiteId:   website.Id,
		Slug:        create.Slug,
		Title:       create.Title,
		Description: create.Description,
		Content:     create.Content,
		Nav:         create.Nav,
		Rank:        int32(pagesCount),
		ImageUrls:   create.ImageUrls,
	}
	if err := dependencies.UnitOfWork.CommercePageRepository().Insert(ctx, &page); err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return dto.NewFailedResponse[*dto_commerce.Page](http.StatusConflict, "page slug is already in use", nil)
		}
		return dto.NewFailedResponse[*dto_commerce.Page](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
	}
	result := dto_commerce.NewPage(page)
	return dto.NewSuccessResponse(&result)
}

func (Create) APISettings() feature.APISettings {
	return feature.NewAPISettings(
		"Create a website page",
		"Create a page for a website in the business account.",
		types.HttpRequestTypeUriJSON,
		http.MethodPost,
		"/v1/commerce/websites/:id/pages",
		true,
		true,
		types.APITagCommerce,
		[]feature.APIError{
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageForbidden, http.StatusForbidden)),
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageUnauthorized, http.StatusUnauthorized)),
			feature.NewAPIError(*exception.NewCustomException("website not found", http.StatusNotFound)),
			feature.NewAPIError(*exception.NewCustomException("page slug is already in use", http.StatusConflict)),
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageInternalServerError, http.StatusInternalServerError)),
		},
	)
}
