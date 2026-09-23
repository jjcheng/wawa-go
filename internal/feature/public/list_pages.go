package feature_public

import (
	"context"
	"net/http"

	"github.com/jjcheng/wawa-go/internal/dto"
	dto_account "github.com/jjcheng/wawa-go/internal/dto/account"
	dto_commerce "github.com/jjcheng/wawa-go/internal/dto/commerce"
	"github.com/jjcheng/wawa-go/internal/exception"
	"github.com/jjcheng/wawa-go/internal/feature"
	"github.com/jjcheng/wawa-go/internal/service"
	"github.com/jjcheng/wawa-go/internal/types"
)

type ListPages struct {
}

func (ListPages) Validate() []exception.InputException {
	return nil
}

func (listPages ListPages) Handle(ctx context.Context, _ *dto_account.User, dependencies *service.Dependencies) dto.Response[[]dto_commerce.Page] {
	if inputErrors := listPages.Validate(); len(inputErrors) > 0 {
		return dto.NewInvalidInputResponse[[]dto_commerce.Page](inputErrors)
	}
	website := getWebsiteFromContext(ctx)
	if website == nil {
		return dto.NewFailedResponse[[]dto_commerce.Page](http.StatusNotFound, "website not found", nil)
	}
	pages, err := dependencies.UnitOfWork.CommercePageRepository().ListByWebsiteId(ctx, website.Id)
	if err != nil {
		return dto.NewFailedResponse[[]dto_commerce.Page](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
	}
	items := make([]dto_commerce.Page, 0, len(pages))
	for _, page := range pages {
		items = append(items, dto_commerce.NewPage(page))
	}
	return dto.NewSuccessResponse(items)
}

func (ListPages) APISettings() feature.APISettings {
	return feature.NewAPISettings(
		"List public website pages",
		"Lists pages for a public website.",
		types.HttpRequestTypeNone,
		http.MethodGet,
		"/v1/public/pages",
		false,
		true,
		types.APITagPublic,
		[]feature.APIError{
			feature.NewAPIError(*exception.NewCustomException("website not found", http.StatusNotFound)),
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageInternalServerError, http.StatusInternalServerError)),
		},
	)
}
