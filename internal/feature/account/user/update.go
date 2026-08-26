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

type Update struct {
	Password        string `json:"password" val:"required" description:"login password"`
	ConfirmPassword string `json:"confirm_password" val:"required" description:"confirm login password"`
	Email           string `json:"email" description:"email of the user"`
	Description     string `json:"description" val:"required" description:"description of the user"`
}

func (update *Update) Validate() []exception.InputException {
	update.Password = strings.TrimSpace(update.Password)
	update.ConfirmPassword = strings.TrimSpace(update.ConfirmPassword)
	update.Email = strings.TrimSpace(update.Email)
	update.Description = strings.TrimSpace(update.Description)
	errors := []exception.InputException{}
	if update.Password == "" {
		errors = append(errors, exception.NewInputException("password", "missing password"))
	} else {
		passwordError := helper.ValidatePassword(update.Password)
		if passwordError != nil {
			errors = append(errors, exception.NewInputException("password", passwordError.Error()))
		} else {
			if update.Password != update.ConfirmPassword {
				errors = append(errors, exception.NewInputException("confirm_password", "password and confirm password must be exactly the same"))
			}
		}
	}
	if update.Email != "" && !helper.ValidateEmail(update.Email) {
		errors = append(errors, exception.NewInputException("email", "invalid email"))
	}
	return errors
}

func (update Update) Handle(ctx context.Context, user *dto_account.User, dependencies *service.Dependencies) dto.Response[*dto_account.User] {
	if errors := update.Validate(); len(errors) > 0 {
		return dto.NewInvalidInputResponse[*dto_account.User](errors)
	}
	existing, ex := dependencies.UnitOfWork.AccountUserRepository().Get(ctx, user.Id)
	if ex != nil {
		return dto.NewFailedResponse[*dto_account.User](ex.StatusCode, ex.Message)
	}
	passwordHash, err := helper.HashPassword(update.Password)
	if err != nil {
		return dto.NewFailedResponse[*dto_account.User](http.StatusInternalServerError, "failed to generate password hash")
	}
	existing.PasswordHash = passwordHash
	existing.Email = update.Email
	existing.Description = update.Description
	if err := dependencies.UnitOfWork.AccountUserRepository().Update(ctx, existing); err != nil {
		dependencies.Logger.ErrorFunction(err, user.Id, update)
		return dto.NewFailedResponse[*dto_account.User](http.StatusInternalServerError, "failed to update user")
	}
	d := dto_account.NewUser(*existing)
	return dto.NewSuccessResponse(&d)
}

func (Update) APISettings() feature.APISettings {
	return feature.NewAPISettings("Update user", "Update the logged in user's password, email, and description", types.HttpRequestTypeJSON, "PATCH", "/account/users/v1", true, false, types.APITagAccount, []feature.APIError{
		feature.NewAPIError(*exception.NewCustomException("failed to generate password hash", http.StatusInternalServerError)),
		feature.NewAPIError(*exception.NewCustomException("failed to update user", http.StatusInternalServerError)),
	})
}
