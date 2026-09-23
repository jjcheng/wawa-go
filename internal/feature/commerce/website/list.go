package feature_commerce_website

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

type List struct {
}

func (List) Validate() []exception.InputException {
	return nil
}

func (list List) Handle(ctx context.Context, user *dto_account.User, dependencies *service.Dependencies) dto.Response[[]dto_commerce.Website] {
	if user == nil {
		return dto.NewFailedResponse[[]dto_commerce.Website](http.StatusForbidden, types.ExceptionMessageForbidden, nil)
	}
	if user.WA == nil || user.WA.BusinessAccount == nil {
		return dto.NewFailedResponse[[]dto_commerce.Website](http.StatusUnauthorized, types.ExceptionMessageUnauthorized, nil)
	}
	if inputErrors := list.Validate(); len(inputErrors) > 0 {
		return dto.NewInvalidInputResponse[[]dto_commerce.Website](inputErrors)
	}
	websites, err := dependencies.UnitOfWork.CommerceWebsiteRepository().ListByBusinessAccountId(ctx, user.WA.BusinessAccount.Id)
	if err != nil {
		return dto.NewFailedResponse[[]dto_commerce.Website](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
	}
	items := make([]dto_commerce.Website, 0, len(websites))
	for _, website := range websites {
		items = append(items, dto_commerce.NewWebsite(website))
	}
	return dto.NewSuccessResponse(items)
}

func (List) APISettings() feature.APISettings {
	return feature.NewAPISettings(
		"List commerce websites",
		"Lists websites stored for the authenticated user's business account.",
		types.HttpRequestTypeNone,
		http.MethodGet,
		"/v1/commerce/websites",
		true,
		true,
		types.APITagCommerce,
		[]feature.APIError{
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageForbidden, http.StatusForbidden)),
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageUnauthorized, http.StatusUnauthorized)),
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageInternalServerError, http.StatusInternalServerError)),
		},
	)
}
