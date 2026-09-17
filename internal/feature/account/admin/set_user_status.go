package feature_account_admin

import (
	"context"
	"errors"
	"net/http"

	"github.com/jjcheng/wawa-go/internal/dto"
	dto_account "github.com/jjcheng/wawa-go/internal/dto/account"
	"github.com/jjcheng/wawa-go/internal/exception"
	"github.com/jjcheng/wawa-go/internal/feature"
	"github.com/jjcheng/wawa-go/internal/service"
	"github.com/jjcheng/wawa-go/internal/types"
	"gorm.io/gorm"
)

type SetUserStatus struct {
	UserId int              `json:"id" val:"required" description:"id of the user"`
	Status types.UserStatus `json:"status" val:"required" description:"new status of the user"`
}

func (setUserStatus *SetUserStatus) Validate() []exception.InputException {
	errors := []exception.InputException{}
	if setUserStatus.UserId <= 0 {
		errors = append(errors, exception.NewInputException("id", "missing user id"))
	}
	switch setUserStatus.Status {
	case "":
		errors = append(errors, exception.NewInputException("status", "missing status"))
	case types.UserStatusPendingPassword:
		errors = append(errors, exception.NewInputException("status", "status must not be PENDING"))
	case types.UserStatusActive, types.UserStatusInactive:
	default:
		errors = append(errors, exception.NewInputException("status", "invalid status"))
	}
	return errors
}

func (setUserStatus SetUserStatus) Handle(ctx context.Context, user *dto_account.User, dependencies *service.Dependencies) dto.Response[any] {
	if user == nil {
		return dto.NewFailedResponse[any](http.StatusForbidden, "you are not authenticated")
	}
	if user.Type != types.UserTypeMaster {
		return dto.NewFailedResponse[any](http.StatusUnauthorized, "you are not master")
	}
	if user.WA == nil {
		return dto.NewFailedResponse[any](http.StatusNotFound, "user's WhatsApp not found")
	}
	if errors := setUserStatus.Validate(); len(errors) > 0 {
		return dto.NewInvalidInputResponse[any](errors)
	}
	if user.Id == int32(setUserStatus.UserId) {
		return dto.NewFailedResponse[any](http.StatusBadRequest, "you cannot update status of yourself")
	}
	existingUser, err := dependencies.UnitOfWork.AccountUserRepository().Get(ctx, int32(setUserStatus.UserId))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return dto.NewFailedResponse[any](http.StatusNotFound, "user not found")
		}
		return dto.NewFailedResponse[any](http.StatusInternalServerError, types.ExceptionMessageInternalServerError)
	}
	if existingUser.Type == types.UserTypeMaster {
		return dto.NewFailedResponse[any](http.StatusUnauthorized, "you are not authorized")
	}
	targetBusinessPortfolio, _, err := dependencies.UnitOfWork.WAPhoneNumberRepository().GetBusinessPortfolioAndAccountByUserId(ctx, int32(setUserStatus.UserId))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return dto.NewFailedResponse[any](http.StatusNotFound, "Meta business portfolio not found")
		}
		return dto.NewFailedResponse[any](http.StatusInternalServerError, types.ExceptionMessageInternalServerError)
	}
	if user.WA.BusinessPortfolio.MetaBusinessPortfolioId != targetBusinessPortfolio.MetaBusinessPortfolioId {
		return dto.NewFailedResponse[any](http.StatusUnauthorized, "you are not authorized to update this user")
	}
	existingUser.Status = setUserStatus.Status
	err = dependencies.UnitOfWork.AccountUserRepository().Update(ctx, existingUser)
	if err != nil {
		return dto.NewFailedResponse[any](http.StatusInternalServerError, types.ExceptionMessageInternalServerError)
	}
	return dto.NewEmptyResponse(true, http.StatusOK)
}

func (SetUserStatus) APISettings() feature.APISettings {
	return feature.NewAPISettings("Admin update user status", "Enable or disable a user. Only admin can update user status.", types.HttpRequestTypeJSON, "PATCH", "/v1/admin/user-status", true, false, types.APITagAccount, []feature.APIError{
		feature.NewAPIError(*exception.NewCustomException("you are not master", http.StatusUnauthorized)),
		feature.NewAPIError(*exception.NewCustomException("you cannot update status of yourself", http.StatusBadRequest)),
		feature.NewAPIError(*exception.NewCustomException("user not found", http.StatusNotFound)),
		feature.NewAPIError(*exception.NewCustomException("you are not authorized to update this user", http.StatusUnauthorized)),
		feature.NewAPIError(*exception.NewCustomException("Meta business portfolio not found", http.StatusNotFound)),
		feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageInternalServerError, http.StatusInternalServerError)),
	})
}
