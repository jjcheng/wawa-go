package feature_account_admin

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

type ListAssignedUsers struct {
	PhoneNumberId int32 `form:"phone_number_id" val:"required" description:"id of the phone number"`
}

func (list *ListAssignedUsers) Validate() []exception.InputException {
	if list.PhoneNumberId <= 0 {
		return []exception.InputException{exception.NewInputException("phone_number_id", "invalid phone number id")}
	}
	return nil
}

func (list ListAssignedUsers) Handle(ctx context.Context, user *dto_account.User, dependencies *service.Dependencies) dto.Response[[]dto_account.UserPhoneNumber] {
	if user == nil {
		return dto.NewFailedResponse[[]dto_account.UserPhoneNumber](http.StatusForbidden, types.ExceptionMessageForbidden, nil)
	}
	if user.Type != types.UserTypeMaster {
		return dto.NewFailedResponse[[]dto_account.UserPhoneNumber](http.StatusUnauthorized, types.ExceptionMessageUnauthorized, nil)
	}
	if inputErrors := list.Validate(); len(inputErrors) > 0 {
		return dto.NewInvalidInputResponse[[]dto_account.UserPhoneNumber](inputErrors)
	}
	phoneNumber, err := dependencies.UnitOfWork.WAPhoneNumberRepository().GetById(ctx, list.PhoneNumberId)
	if err != nil {
		return dto.NewFailedResponse[[]dto_account.UserPhoneNumber](http.StatusNotFound, "phone number not found", nil)
	}
	if phoneNumber.BusinessAccountId != user.BusinessAccountId {
		return dto.NewFailedResponse[[]dto_account.UserPhoneNumber](http.StatusUnauthorized, types.ExceptionMessageUnauthorized, nil)
	}
	assignments, err := dependencies.UnitOfWork.AccountUserPhoneNumberRepository().ListByPhoneNumberId(ctx, list.PhoneNumberId)
	if err != nil {
		return dto.NewFailedResponse[[]dto_account.UserPhoneNumber](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
	}
	items := make([]dto_account.UserPhoneNumber, 0, len(assignments))
	for index := range assignments {
		items = append(items, dto_account.NewUserPhoneNumber(&assignments[index]))
	}
	return dto.NewSuccessResponse(items)
}

func (ListAssignedUsers) APISettings() feature.APISettings {
	return feature.NewAPISettings(
		"List users assigned to phone number",
		"Lists users assigned to a phone number in the authenticated business account.",
		types.HttpRequestTypeQuery,
		http.MethodGet,
		"/v1/admin/assigned-users",
		true,
		true,
		types.APITagAccount,
		[]feature.APIError{
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageForbidden, http.StatusForbidden)),
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageUnauthorized, http.StatusUnauthorized)),
			feature.NewAPIError(*exception.NewCustomException("phone number not found", http.StatusNotFound)),
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageInternalServerError, http.StatusInternalServerError)),
		},
	)
}
