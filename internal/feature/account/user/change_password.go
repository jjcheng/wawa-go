package feature_account_user

import (
	"context"
	"net/http"
	"strings"

	"github.com/jjcheng/wawa-go/internal/dto"
	dto_account "github.com/jjcheng/wawa-go/internal/dto/account"
	"github.com/jjcheng/wawa-go/internal/exception"
	"github.com/jjcheng/wawa-go/internal/feature"
	"github.com/jjcheng/wawa-go/internal/helper"
	"github.com/jjcheng/wawa-go/internal/service"
	"github.com/jjcheng/wawa-go/internal/types"
)

type ChangePassword struct {
	OldPassword        string `json:"old_password" val:"required" description:"current password"`
	NewPassword        string `json:"new_password" val:"required" description:"new password"`
	ConfirmNewPassword string `json:"confirm_new_password" val:"required" description:"confirm new password"`
}

func (changePassword *ChangePassword) Validate() []exception.InputException {
	changePassword.OldPassword = strings.TrimSpace(changePassword.OldPassword)
	changePassword.NewPassword = strings.TrimSpace(changePassword.NewPassword)
	changePassword.ConfirmNewPassword = strings.TrimSpace(changePassword.ConfirmNewPassword)
	errors := []exception.InputException{}
	if changePassword.OldPassword == "" {
		errors = append(errors, exception.NewInputException("old_password", "missing old password"))
	} else if passwordError := helper.ValidatePassword(changePassword.OldPassword); passwordError != nil {
		errors = append(errors, exception.NewInputException("old_password", passwordError.Error()))
	}
	if changePassword.NewPassword == "" {
		errors = append(errors, exception.NewInputException("new_password", "missing new password"))
	} else if passwordError := helper.ValidatePassword(changePassword.NewPassword); passwordError != nil {
		errors = append(errors, exception.NewInputException("new_password", passwordError.Error()))
	} else if changePassword.NewPassword == changePassword.OldPassword {
		errors = append(errors, exception.NewInputException("new_password", "new password cannot be the same as old password"))
	} else if changePassword.NewPassword != changePassword.ConfirmNewPassword {
		errors = append(errors, exception.NewInputException("confirm_new_password", "new password and confirm new password must be exactly the same"))
	}
	return errors
}

func (changePassword ChangePassword) Handle(ctx context.Context, user *dto_account.User, dependencies *service.Dependencies) dto.Response[*dto_account.User] {
	if errors := changePassword.Validate(); len(errors) > 0 {
		return dto.NewInvalidInputResponse[*dto_account.User](errors)
	}
	existing, ex := dependencies.UnitOfWork.AccountUserRepository().Get(ctx, user.Id)
	if ex != nil {
		return dto.NewFailedResponse[*dto_account.User](ex.StatusCode, ex.Message)
	}
	if !helper.VerifyPassword(changePassword.OldPassword, existing.PasswordHash) {
		return dto.NewFailedResponse[*dto_account.User](http.StatusUnauthorized, "invalid old password")
	}
	passwordHash, err := helper.HashPassword(changePassword.NewPassword)
	if err != nil {
		return dto.NewFailedResponse[*dto_account.User](http.StatusInternalServerError, "failed to generate password hash")
	}
	existing.PasswordHash = passwordHash
	if err := dependencies.UnitOfWork.AccountUserRepository().Update(ctx, existing); err != nil {
		dependencies.Logger.ErrorFunction(err, user.Id, changePassword)
		return dto.NewFailedResponse[*dto_account.User](http.StatusInternalServerError, "failed to change password")
	}
	d := dto_account.NewUser(*existing)
	return dto.NewSuccessResponse(&d)
}

func (ChangePassword) APISettings() feature.APISettings {
	return feature.NewAPISettings("Change password", "Change the logged in user's password", types.HttpRequestTypeJSON, http.MethodPatch, "/account/users/v1/password", true, true, types.APITagAccount, []feature.APIError{
		feature.NewAPIError(*exception.NewCustomException("invalid old password", http.StatusUnauthorized)),
		feature.NewAPIError(*exception.NewCustomException("failed to generate password hash", http.StatusInternalServerError)),
		feature.NewAPIError(*exception.NewCustomException("failed to change password", http.StatusInternalServerError)),
	})
}
