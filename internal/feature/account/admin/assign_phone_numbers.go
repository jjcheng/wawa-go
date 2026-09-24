package feature_account_admin

import (
	"context"
	"errors"
	"net/http"

	dao_account "github.com/jjcheng/wawa-go/internal/dao/account"
	"github.com/jjcheng/wawa-go/internal/dto"
	dto_account "github.com/jjcheng/wawa-go/internal/dto/account"
	"github.com/jjcheng/wawa-go/internal/exception"
	"github.com/jjcheng/wawa-go/internal/feature"
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
	if user.WA == nil || user.WA.BusinessAccount == nil {
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
	if err := transaction.AccountUserPhoneNumberRepository().DeleteByUserId(ctx, user.Id); err != nil {
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
	return feature.NewAPISettings(
		"Assign phone numbers to user",
		"Replaces the phone numbers managed by a user in the authenticated business account.",
		types.HttpRequestTypeJSON,
		http.MethodPatch,
		"/v1/admin/assign-phone-numbers",
		true,
		true,
		types.APITagAccount,
		[]feature.APIError{
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageForbidden, http.StatusForbidden)),
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageUnauthorized, http.StatusUnauthorized)),
			feature.NewAPIError(*exception.NewCustomException("user not found", http.StatusNotFound)),
			feature.NewAPIError(*exception.NewCustomException("one or more of the phone numbers are unavailable", http.StatusBadRequest)),
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageInternalServerError, http.StatusInternalServerError)),
		},
	)
}
