package feature_account_admin

import (
	"context"
	"net/http"

	"github.com/jjcheng/wawa-go/internal/dto"
	dto_account "github.com/jjcheng/wawa-go/internal/dto/account"
	dto_ai "github.com/jjcheng/wawa-go/internal/dto/ai"
	"github.com/jjcheng/wawa-go/internal/exception"
	"github.com/jjcheng/wawa-go/internal/feature"
	feature_wa_phone_number "github.com/jjcheng/wawa-go/internal/feature/wa/phone_number"
	"github.com/jjcheng/wawa-go/internal/service"
	"github.com/jjcheng/wawa-go/internal/types"
)

type ListAssignedUsers struct {
	PhoneNumberId int32 `form:"phone_number_id" json:"phone_number_id" val:"required" description:"id of the phone number"`
}

func (list *ListAssignedUsers) Validate() []exception.InputException {
	if list.PhoneNumberId <= 0 {
		return []exception.InputException{exception.NewInputException("phone_number_id", "invalid phone number id")}
	}
	return nil
}

func (list ListAssignedUsers) Handle(ctx context.Context, user *dto_account.User, dependencies *service.Dependencies) dto.Response[[]dto_account.User] {
	if user == nil {
		return dto.NewFailedResponse[[]dto_account.User](http.StatusForbidden, types.ExceptionMessageForbidden, nil)
	}
	if user.Type != types.UserTypeMaster {
		return dto.NewFailedResponse[[]dto_account.User](http.StatusUnauthorized, types.ExceptionMessageUnauthorized, nil)
	}
	if inputErrors := list.Validate(); len(inputErrors) > 0 {
		return dto.NewInvalidInputResponse[[]dto_account.User](inputErrors)
	}
	phoneNumber, err := dependencies.UnitOfWork.WAPhoneNumberRepository().GetById(ctx, list.PhoneNumberId)
	if err != nil {
		return dto.NewFailedResponse[[]dto_account.User](http.StatusNotFound, "phone number not found", nil)
	}
	if phoneNumber.BusinessAccountId != user.BusinessAccountId {
		return dto.NewFailedResponse[[]dto_account.User](http.StatusUnauthorized, types.ExceptionMessageUnauthorized, nil)
	}
	assignments, err := dependencies.UnitOfWork.AccountUserPhoneNumberRepository().ListByPhoneNumberId(ctx, list.PhoneNumberId)
	if err != nil {
		return dto.NewFailedResponse[[]dto_account.User](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
	}
	var users []dto_account.User
	for _, assignment := range assignments {
		users = append(users, dto_account.NewUser(*assignment.User))
	}
	// items := make([]dto_account.UserPhoneNumber, 0, len(assignments))
	// for index := range assignments {
	// 	items = append(items, dto_account.NewUserPhoneNumber(&assignments[index]))
	// }
	return dto.NewSuccessResponse(users)
}

func (ListAssignedUsers) APISettings() feature.APISettings {
	return feature.NewAPISettings(
		"List users assigned to a phone number",
		"List all users assigned to a phone number. Only master users can access this endpoint",
		types.HttpRequestTypeQuery,
		http.MethodGet,
		"/v1/admin/assigned-users",
		true,
		true,
		types.APITagAdmin,
		[]feature.APIError{
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageForbidden, http.StatusForbidden)),
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageUnauthorized, http.StatusUnauthorized)),
			feature.NewAPIError(*exception.NewCustomException("phone number not found", http.StatusNotFound)),
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageInternalServerError, http.StatusInternalServerError)),
		},
		feature.NewAIWorker(true,
			"To view users assigned to a phone number, go to Assets -> Phone numbers and select the phone number.",
			types.AIWorkerReturnTypeData,
			"",
			"/assets/phone-numbers",
			feature.NewAIWorkerRequire("Select a connected phone number", feature_wa_phone_number.List{Status: types.WAPhoneNumberStatusConnected}, dto_ai.WorkInput{
				Name:               "phone_number_id",
				Description:        "phone number to view assigned users for",
				Type:               types.AIInputFieldTypeInt,
				ReferenceFieldName: "id",
				DisplayType:        types.AIWorkerDisplayTypeSingleChoiceTable,
			}),
		),
	)
}
