package feature_account_admin

import (
	"context"
	"errors"
	"net/http"

	dao_account "github.com/jjcheng/wawa-go/internal/dao/account"
	"github.com/jjcheng/wawa-go/internal/dto"
	dto_account "github.com/jjcheng/wawa-go/internal/dto/account"
	dto_ai_worker "github.com/jjcheng/wawa-go/internal/dto/ai_worker"
	"github.com/jjcheng/wawa-go/internal/exception"
	"github.com/jjcheng/wawa-go/internal/feature"
	feature_wa_phone_number "github.com/jjcheng/wawa-go/internal/feature/wa/phone_number"
	"github.com/jjcheng/wawa-go/internal/service"
	"github.com/jjcheng/wawa-go/internal/types"
	"gorm.io/gorm"
)

type AssignPhoneNumbers struct {
	UserId         int32   `json:"user_id" val:"required" description:"id of the user"`
	PhoneNumberIds []int32 `json:"phone_number_ids" val:"required" description:"phone number ids to assign to the user"` // can be 0
}

func (assignPhoneNumbers *AssignPhoneNumbers) Validate() []exception.InputException {
	var errors []exception.InputException
	if assignPhoneNumbers.UserId <= 0 {
		errors = append(errors, exception.NewInputException("user_id", "missing user_id"))
	}
	return errors
}

func (assignPhoneNumbers AssignPhoneNumbers) Handle(ctx context.Context, user *dto_account.User, dependencies *service.Dependencies) dto.Response[any] {
	if user == nil {
		return dto.NewFailedResponse[any](http.StatusForbidden, types.ExceptionMessageForbidden, nil)
	}
	if user.Type != types.UserTypeMaster {
		return dto.NewFailedResponse[any](http.StatusUnauthorized, types.ExceptionMessageUnauthorized, nil)
	}
	targetUser, err := dependencies.UnitOfWork.AccountUserRepository().GetById(ctx, assignPhoneNumbers.UserId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return dto.NewFailedResponse[any](http.StatusNotFound, "user not found", nil)
		}
		return dto.NewFailedResponse[any](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
	}
	if targetUser.BusinessAccountId != user.BusinessAccountId {
		return dto.NewFailedResponse[any](http.StatusUnauthorized, types.ExceptionMessageUnauthorized, nil)
	}
	// if there is phoneNumberIds, check these phone number ids belong to the business account
	if len(assignPhoneNumbers.PhoneNumberIds) > 0 {
		count, err := dependencies.UnitOfWork.AccountUserPhoneNumberRepository().CountByBusinessAccountIdAndPhoneNumberIds(ctx, user.BusinessAccountId, assignPhoneNumbers.PhoneNumberIds)
		if err != nil {
			return dto.NewFailedResponse[any](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
		}
		if count != len(assignPhoneNumbers.PhoneNumberIds) {
			return dto.NewFailedResponse[any](http.StatusBadRequest, "one or more of the phone numbers are unavailable", nil)
		}
	}
	var committed bool
	transaction := dependencies.UnitOfWork.BeginTransaction()
	defer func() {
		if committed {
			transaction.Rollback()
		}
	}()
	// delete existing assignments
	if err := transaction.AccountUserPhoneNumberRepository().DeleteByUserId(ctx, assignPhoneNumbers.UserId); err != nil {
		return dto.NewFailedResponse[any](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
	}
	// insert new assignments
	for _, phoneNumberId := range assignPhoneNumbers.PhoneNumberIds {
		userPhoneNumber := dao_account.UserPhoneNumber{
			UserId:        assignPhoneNumbers.UserId,
			PhoneNumberId: phoneNumberId,
		}
		if err := transaction.AccountUserPhoneNumberRepository().Insert(ctx, &userPhoneNumber); err != nil {
			return dto.NewFailedResponse[any](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
		}
	}
	// commit
	if err := transaction.CommitTransaction(); err != nil {
		return dto.NewFailedResponse[any](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
	}
	committed = true
	return dto.NewEmptyResponse(true, http.StatusOK)
}

func (AssignPhoneNumbers) APISettings() feature.APISettings {
	apiSettings := feature.NewAPISettings(
		"Assign phone numbers to a user",
		"Assign or remove the phone numbers managed by a user. Only master users can access this endpoint",
		types.HttpRequestTypeJSON,
		http.MethodPost,
		"/v1/admin/assign-phone-numbers",
		true,
		true,
		types.APITagAdmin,
		[]feature.APIError{
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageForbidden, http.StatusForbidden)),
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageUnauthorized, http.StatusUnauthorized)),
			feature.NewAPIError(*exception.NewCustomException("user not found", http.StatusNotFound)),
			feature.NewAPIError(*exception.NewCustomException("one or more of the phone numbers are unavailable", http.StatusBadRequest)),
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageInternalServerError, http.StatusInternalServerError)),
		}, nil,
	)
	aiWorker := feature.NewAIWorker(
		true,
		"To assign phone numbers to a user, go to Assets -> Users, select a user, click Phone numbers, select phone numbers you want to assign to the user, click Save.",
		types.AIWorkerReturnTypeText,
		"Phone numbers successfully assigned to user.",
		"/assets/users",
		feature.NewAIWorkerRequire(
			"Please select a user to assign phone numbers to",
			ListUsers{},
			dto_ai_worker.WorkInput{
				Name:               "user_id",
				Description:        "id of the user",
				Type:               types.AIWorkerInputFieldTypeInt,
				ReferenceFieldName: "id",
				DisplayType:        types.AIWorkerDisplayTypeSingleChoiceTable,
			},
		),
		feature.NewAIWorkerRequire(
			"Please select phone numbers to assign to this user",
			feature_wa_phone_number.List{Status: types.WAPhoneNumberStatusConnected},
			dto_ai_worker.WorkInput{
				Name:               "phone_number_ids",
				Description:        "id of the phone number",
				Type:               types.AIWorkerInputFieldTypeInt,
				ReferenceFieldName: "id",
				DisplayType:        types.AIWorkerDisplayTypeMultiChoiceTable,
			},
		),
	)
	apiSettings.AIWorker = aiWorker
	return apiSettings
}
