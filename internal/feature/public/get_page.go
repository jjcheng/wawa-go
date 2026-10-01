package feature_public

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

type GetPage struct {
	Slug string `form:"slug" val:"required" description:"slug of the page"`
}

func (getPage *GetPage) Validate() []exception.InputException {
	getPage.Slug = strings.TrimSpace(getPage.Slug)
	if getPage.Slug == "" {
		return []exception.InputException{exception.NewInputException("slug", "invalid slug")}
	}
	return nil
}

func (getPage GetPage) Handle(ctx context.Context, _ *dto_account.User, dependencies *service.Dependencies) dto.Response[*dto_commerce.Page] {
	if inputErrors := getPage.Validate(); len(inputErrors) > 0 {
		return dto.NewInvalidInputResponse[*dto_commerce.Page](inputErrors)
	}
	website := getWebsiteFromContext(ctx)
	if website == nil {
		return dto.NewFailedResponse[*dto_commerce.Page](http.StatusNotFound, "website not found", nil)
	}
	page, err := dependencies.UnitOfWork.CommercePageRepository().GetBySlug(ctx, website.Id, getPage.Slug)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return dto.NewFailedResponse[*dto_commerce.Page](http.StatusNotFound, "page not found", nil)
		}
		return dto.NewFailedResponse[*dto_commerce.Page](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
	}
	if page.WebsiteId != website.Id {
		return dto.NewFailedResponse[*dto_commerce.Page](http.StatusUnauthorized, types.ExceptionMessageUnauthorized, nil)
	}
	result := dto_commerce.NewPage(*page)
	return dto.NewSuccessResponse(&result)
}

func (GetPage) APISettings() feature.APISettings {
	return feature.NewAPISettings(
		"Get a public website page",
		"Get a page from a public website.",
		types.HttpRequestTypeQuery,
		http.MethodGet,
		"/v1/public/page",
		false,
		true,
		types.APITagPublic,
		[]feature.APIError{
			feature.NewAPIError(*exception.NewCustomException("website not found", http.StatusNotFound)),
			feature.NewAPIError(*exception.NewCustomException("page not found", http.StatusNotFound)),
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageInternalServerError, http.StatusInternalServerError)),
		},
	)
}
