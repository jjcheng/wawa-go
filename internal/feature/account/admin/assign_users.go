package feature_account_admin

import (
	"context"
	"net/http"

	dao_account "github.com/jjcheng/wawa-go/internal/dao/account"
	"github.com/jjcheng/wawa-go/internal/dto"
	dto_account "github.com/jjcheng/wawa-go/internal/dto/account"
	dto_ai "github.com/jjcheng/wawa-go/internal/dto/ai"
	"github.com/jjcheng/wawa-go/internal/exception"
	"github.com/jjcheng/wawa-go/internal/feature"
	feature_wa_phone_number "github.com/jjcheng/wawa-go/internal/feature/wa/phone_number"
	"github.com/jjcheng/wawa-go/internal/service"
	"github.com/jjcheng/wawa-go/internal/types"
	"gorm.io/gorm"
)

type AssignUsers struct {
	PhoneNumberId int32   `json:"phone_number_id" val:"required" description:"phone number id to assign users to"`
	UserIds       []int32 `json:"user_ids" val:"required" description:"user ids to assign this phone number to"`
}

func (assignUsers *AssignUsers) Validate() []exception.InputException {
	inputErrors := []exception.InputException{}
	if assignUsers.PhoneNumberId <= 0 {
		inputErrors = append(inputErrors, exception.NewInputException("phone_number_id", "invalid phone number id"))
	}
	if len(assignUsers.UserIds) == 0 {
		inputErrors = append(inputErrors, exception.NewInputException("user_ids", "missing user ids"))
	}
	return inputErrors
}

func (assignUsers AssignUsers) Handle(ctx context.Context, user *dto_account.User, dependencies *service.Dependencies) dto.Response[any] {
	if user == nil {
		return dto.NewFailedResponse[any](http.StatusForbidden, types.ExceptionMessageForbidden, nil)
	}
	if user.Type != types.UserTypeMaster {
		return dto.NewFailedResponse[any](http.StatusUnauthorized, types.ExceptionMessageUnauthorized, nil)
	}
	if inputErrors := assignUsers.Validate(); len(inputErrors) > 0 {
		return dto.NewInvalidInputResponse[any](inputErrors)
	}
	phoneNumber, err := dependencies.UnitOfWork.WAPhoneNumberRepository().GetById(ctx, assignUsers.PhoneNumberId)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return dto.NewFailedResponse[any](http.StatusNotFound, "phone number not found", nil)
		}
		return dto.NewFailedResponse[any](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
	}
	if phoneNumber.BusinessAccountId != user.BusinessAccountId {
		return dto.NewFailedResponse[any](http.StatusUnauthorized, types.ExceptionMessageUnauthorized, nil)
	}
	users, err := dependencies.UnitOfWork.AccountUserRepository().GetByIdsAndBusinessAccountId(ctx, assignUsers.UserIds, user.BusinessAccountId)
	if err != nil {
		return dto.NewFailedResponse[any](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
	}
	if len(users) != len(assignUsers.UserIds) {
		return dto.NewFailedResponse[any](http.StatusBadRequest, "one or more users are unavailable", nil)
	}
	transaction := dependencies.UnitOfWork.BeginTransaction()
	committed := false
	defer func() {
		if !committed {
			transaction.Rollback()
		}
	}()
	if err := transaction.AccountUserPhoneNumberRepository().DeleteByPhoneNumberId(ctx, assignUsers.PhoneNumberId); err != nil {
		return dto.NewFailedResponse[any](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
	}
	for _, userId := range assignUsers.UserIds {
		assignment := dao_account.UserPhoneNumber{UserId: userId, PhoneNumberId: assignUsers.PhoneNumberId}
		if err := transaction.AccountUserPhoneNumberRepository().Insert(ctx, &assignment); err != nil {
			return dto.NewFailedResponse[any](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
		}
	}
	if err := transaction.CommitTransaction(); err != nil {
		return dto.NewFailedResponse[any](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
	}
	committed = true
	return dto.NewEmptyResponse(true, http.StatusOK)
}

func (AssignUsers) APISettings() feature.APISettings {
	return feature.NewAPISettings(
		"Assign users to a phone number",
		"Assign or remove the users managing a phone number. Only master users can access this endpoint",
		types.HttpRequestTypeJSON,
		http.MethodPost,
		"/v1/admin/assign-users",
		true,
		true,
		types.APITagAdmin,
		[]feature.APIError{
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageForbidden, http.StatusForbidden)),
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageUnauthorized, http.StatusUnauthorized)),
			feature.NewAPIError(*exception.NewCustomException("phone number not found", http.StatusNotFound)),
			feature.NewAPIError(*exception.NewCustomException("one or more users are unavailable", http.StatusBadRequest)),
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageInternalServerError, http.StatusInternalServerError)),
		},
		feature.NewAIWorker(
			true,
			"To assign users to a phone number, go to Assets -> Phone numbers, select a phone number. In the pop up, click Manage button, then choose the users who you want to assign it to. This feature is limited to MASTER users",
			types.AIWorkerReturnTypeText,
			"Users has successfully assigned to the phone number.",
			"/assets/phone-numbers",
			feature.NewAIWorkerRequire(
				"Select a connected phone number",
				feature_wa_phone_number.List{Status: types.WAPhoneNumberStatusConnected},
				dto_ai.WorkInput{
					Name:               "phone_number_id",
					Description:        "phone number to assign users to",
					Type:               types.AIInputFieldTypeInt,
					ReferenceFieldName: "id",
					DisplayType:        types.AIWorkerDisplayTypeSingleChoiceTable,
				},
			),
			feature.NewAIWorkerRequire(
				"Select the users to assign to this phone number",
				ListUsers{},
				dto_ai.WorkInput{
					Name:               "user_ids",
					Description:        "users to assign to the phone number",
					Type:               types.AIInputFieldTypeInt,
					ReferenceFieldName: "id",
					DisplayType:        types.AIWorkerDisplayTypeMultiChoiceTable,
				},
			),
		),
	)
}
