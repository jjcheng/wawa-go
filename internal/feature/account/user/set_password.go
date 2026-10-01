package feature_account_user

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/jjcheng/wawa-go/internal/dto"
	dto_account "github.com/jjcheng/wawa-go/internal/dto/account"
	"github.com/jjcheng/wawa-go/internal/exception"
	"github.com/jjcheng/wawa-go/internal/feature"
	"github.com/jjcheng/wawa-go/internal/helper"
	"github.com/jjcheng/wawa-go/internal/service"
	"github.com/jjcheng/wawa-go/internal/types"
	"gorm.io/gorm"
)

// SetPassword lets a user created by embedded signup choose their first password. The temporary password
// generated during signup is random and never disclosed, so the session issued at signup is the only proof of identity.
type SetPassword struct {
	NewPassword        string `json:"new_password" val:"required" description:"new password"`
	ConfirmNewPassword string `json:"confirm_new_password" val:"required" description:"confirm new password"`
}

func (setPassword *SetPassword) Validate() []exception.InputException {
	setPassword.NewPassword = strings.TrimSpace(setPassword.NewPassword)
	setPassword.ConfirmNewPassword = strings.TrimSpace(setPassword.ConfirmNewPassword)
	errors := []exception.InputException{}
	if setPassword.NewPassword == "" {
		errors = append(errors, exception.NewInputException("new_password", "missing new password"))
	} else if passwordError := helper.ValidatePassword(setPassword.NewPassword); passwordError != nil {
		errors = append(errors, exception.NewInputException("new_password", passwordError.Error()))
	} else if setPassword.NewPassword != setPassword.ConfirmNewPassword {
		errors = append(errors, exception.NewInputException("confirm_new_password", "new password and confirm new password must be exactly the same"))
	}
	return errors
}

func (setPassword SetPassword) Handle(ctx context.Context, user *dto_account.User, dependencies *service.Dependencies) dto.Response[*dto_account.User] {
	if user == nil {
		return dto.NewFailedResponse[*dto_account.User](http.StatusForbidden, types.ExceptionMessageForbidden, nil)
	}
	if errors := setPassword.Validate(); len(errors) > 0 {
		return dto.NewInvalidInputResponse[*dto_account.User](errors)
	}
	// once a password is set this path must not be reusable, otherwise a stolen session could reset it without the current password
	if user.Status != types.UserStatusPendingPassword {
		return dto.NewFailedResponse[*dto_account.User](http.StatusConflict, "password is already set, use change password instead", nil)
	}
	existing, err := dependencies.UnitOfWork.AccountUserRepository().GetById(ctx, user.Id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return dto.NewFailedResponse[*dto_account.User](http.StatusNotFound, "user not found", nil)
		}
		return dto.NewFailedResponse[*dto_account.User](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
	}
	passwordHash, err := helper.HashPassword(setPassword.NewPassword)
	if err != nil {
		return dto.NewFailedResponse[*dto_account.User](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
	}
	existing.PasswordHash = passwordHash
	existing.Status = types.UserStatusActive
	if err := dependencies.UnitOfWork.AccountUserRepository().Update(ctx, existing); err != nil {
		return dto.NewFailedResponse[*dto_account.User](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
	}
	d := dto_account.NewUser(*existing)
	return dto.NewSuccessResponse(&d)
}

func (SetPassword) APISettings() feature.APISettings {
	return feature.NewAPISettings("Set initial password", "Set the first password for a user created by WhatsApp embedded signup", types.HttpRequestTypeJSON, http.MethodPatch, "/v1/account/users/me/initial-password", true, true, types.APITagUser, []feature.APIError{
		feature.NewAPIError(*exception.NewCustomException("password is already set, use change password instead", http.StatusConflict)),
		feature.NewAPIError(*exception.NewCustomException("user not found", http.StatusNotFound)),
		feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageInternalServerError, http.StatusInternalServerError)),
	})
}
