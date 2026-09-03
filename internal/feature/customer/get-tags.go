package feature_customer

import (
	"context"
	"net/http"

	"github.com/jjcheng/wawa-go/internal/dto"
	dto_account "github.com/jjcheng/wawa-go/internal/dto/account"
	"github.com/jjcheng/wawa-go/internal/exception"
	"github.com/jjcheng/wawa-go/internal/feature"
	"github.com/jjcheng/wawa-go/internal/service"
	"github.com/jjcheng/wawa-go/internal/types"
)

type GetTags struct {
}

func (getTags *GetTags) Validate() []exception.InputException {
	return []exception.InputException{}
}

func (getTags GetTags) Handle(ctx context.Context, user *dto_account.User, dependencies *service.Dependencies) dto.Response[[]string] {
	if inputErrors := getTags.Validate(); len(inputErrors) > 0 {
		return dto.NewInvalidInputResponse[[]string](inputErrors)
	}
	tags, err := dependencies.UnitOfWork.CustomerRepository().GetDistinctTags(ctx, user.Id)
	if err != nil {
		return dto.NewFailedResponse[[]string](http.StatusInternalServerError, types.ExceptionMessageInternalServerError)
	}
	return dto.NewSuccessResponse(tags)
}

func (GetTags) APISettings() feature.APISettings {
	return feature.NewAPISettings(
		"Get customer tags",
		"Lists distinct customer tags for the authenticated user.",
		types.HttpRequestTypeNone,
		http.MethodGet,
		"/v1/customers/tags",
		true,
		true,
		types.APITagCustomer,
		[]feature.APIError{
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageInternalServerError, http.StatusInternalServerError)),
		},
	)
}
