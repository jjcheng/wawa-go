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

type List struct {
	WebsiteId int32 `uri:"id" val:"required" description:"id of the website"`
}

func (list *List) Validate() []exception.InputException {
	if list.WebsiteId <= 0 {
		return []exception.InputException{exception.NewInputException("id", "invalid website id")}
	}
	return nil
}

func (list List) Handle(ctx context.Context, user *dto_account.User, dependencies *service.Dependencies) dto.Response[[]dto_commerce.Page] {
	if user == nil {
		return dto.NewFailedResponse[[]dto_commerce.Page](http.StatusForbidden, types.ExceptionMessageForbidden, nil)
	}
	if inputErrors := list.Validate(); len(inputErrors) > 0 {
		return dto.NewInvalidInputResponse[[]dto_commerce.Page](inputErrors)
	}
	website, err := dependencies.UnitOfWork.CommerceWebsiteRepository().GetById(ctx, list.WebsiteId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return dto.NewFailedResponse[[]dto_commerce.Page](http.StatusNotFound, "website not found", nil)
		}
		return dto.NewFailedResponse[[]dto_commerce.Page](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
	}
	if website.BusinessAccountId != user.WA.BusinessAccount.Id {
		return dto.NewFailedResponse[[]dto_commerce.Page](http.StatusUnauthorized, types.ExceptionMessageUnauthorized, nil)
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

func (List) APISettings() feature.APISettings {
	return feature.NewAPISettings(
		"List commerce website pages",
		"Lists pages belonging to a website in the authenticated user's business account.",
		types.HttpRequestTypeUri,
		http.MethodGet,
		"/v1/commerce/websites/:id/pages",
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
