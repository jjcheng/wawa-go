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

type ListTags struct {
}

func (listTags *ListTags) Validate() []exception.InputException {
	return nil
}

func (listTags ListTags) Handle(ctx context.Context, user *dto_account.User, dependencies *service.Dependencies) dto.Response[[]string] {
	if user == nil {
		return dto.NewFailedResponse[[]string](http.StatusForbidden, types.ExceptionMessageForbidden, nil)
	}
	if user.WA == nil {
		return dto.NewFailedResponse[[]string](http.StatusBadRequest, "no phone number has been assigned to you", nil)
	}
	if inputErrors := listTags.Validate(); len(inputErrors) > 0 {
		return dto.NewInvalidInputResponse[[]string](inputErrors)
	}
	tags, err := dependencies.UnitOfWork.CustomerRepository().GetDistinctTagsByPhoneNumberIds(ctx, user.WA.PhoneNumberIds())
	if err != nil {
		return dto.NewFailedResponse[[]string](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
	}
	return dto.NewSuccessResponse(tags)
}

func (ListTags) APISettings() feature.APISettings {
	return feature.NewAPISettings(
		"List customer tags",
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
