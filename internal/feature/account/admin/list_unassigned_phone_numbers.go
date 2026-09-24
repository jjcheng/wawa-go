package feature_account_admin

import (
	"context"
	"net/http"

	"github.com/jjcheng/wawa-go/internal/dto"
	dto_account "github.com/jjcheng/wawa-go/internal/dto/account"
	dto_wa "github.com/jjcheng/wawa-go/internal/dto/wa"
	"github.com/jjcheng/wawa-go/internal/exception"
	"github.com/jjcheng/wawa-go/internal/feature"
	"github.com/jjcheng/wawa-go/internal/service"
	"github.com/jjcheng/wawa-go/internal/types"
)

type ListUnassignedPhoneNumbers struct {
}

func (ListUnassignedPhoneNumbers) Validate() []exception.InputException {
	return nil
}

func (list ListUnassignedPhoneNumbers) Handle(ctx context.Context, user *dto_account.User, dependencies *service.Dependencies) dto.Response[[]dto_wa.PhoneNumber] {
	if user == nil {
		return dto.NewFailedResponse[[]dto_wa.PhoneNumber](http.StatusForbidden, types.ExceptionMessageForbidden, nil)
	}
	if user.Type != types.UserTypeMaster || user.WA == nil || user.WA.BusinessAccount == nil {
		return dto.NewFailedResponse[[]dto_wa.PhoneNumber](http.StatusUnauthorized, types.ExceptionMessageUnauthorized, nil)
	}
	if inputErrors := list.Validate(); len(inputErrors) > 0 {
		return dto.NewInvalidInputResponse[[]dto_wa.PhoneNumber](inputErrors)
	}
	phoneNumbers, err := dependencies.UnitOfWork.WAPhoneNumberRepository().ListUnassigned(ctx, user.WA.BusinessAccount.Id)
	if err != nil {
		return dto.NewFailedResponse[[]dto_wa.PhoneNumber](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
	}
	items := make([]dto_wa.PhoneNumber, 0, len(phoneNumbers))
	for _, phoneNumber := range phoneNumbers {
		items = append(items, dto_wa.NewPhoneNumber(phoneNumber))
	}
	return dto.NewSuccessResponse(items)
}

func (ListUnassignedPhoneNumbers) APISettings() feature.APISettings {
	return feature.NewAPISettings(
		"List unassigned WhatsApp phone numbers",
		"Lists WhatsApp phone numbers in the business account that are not assigned to a user.",
		types.HttpRequestTypeNone,
		http.MethodGet,
		"/v1/admin/unassigned-phone-numbers",
		true,
		true,
		types.APITagAccount,
		[]feature.APIError{
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageForbidden, http.StatusForbidden)),
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageUnauthorized, http.StatusUnauthorized)),
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageInternalServerError, http.StatusInternalServerError)),
		},
	)
}
