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

type UpdateProfile struct {
	Name        string `json:"name" val:"required" description:"name of the user"`
	Email       string `json:"email" description:"email of the user"`
	Description string `json:"description" val:"required" description:"description of the user"`
}

func (update *UpdateProfile) Validate() []exception.InputException {
	update.Name = strings.TrimSpace(update.Name)
	update.Email = strings.TrimSpace(update.Email)
	update.Description = strings.TrimSpace(update.Description)
	errors := []exception.InputException{}
	if update.Email != "" && !helper.ValidateEmail(update.Email) {
		errors = append(errors, exception.NewInputException("email", "invalid email"))
	}
	return errors
}

func (update UpdateProfile) Handle(ctx context.Context, user *dto_account.User, dependencies *service.Dependencies) dto.Response[*dto_account.User] {
	if user == nil {
		return dto.NewFailedResponse[*dto_account.User](http.StatusForbidden, types.ExceptionMessageForbidden, nil)
	}
	if errors := update.Validate(); len(errors) > 0 {
		return dto.NewInvalidInputResponse[*dto_account.User](errors)
	}
	existing, err := dependencies.UnitOfWork.AccountUserRepository().GetById(ctx, user.Id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return dto.NewFailedResponse[*dto_account.User](http.StatusNotFound, "user not found", nil)
		}
		return dto.NewFailedResponse[*dto_account.User](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
	}
	if update.Email != "" {
		// check email not du
		existingEmail, err := dependencies.UnitOfWork.AccountUserRepository().GetByEmailAndBusinessAccountId(ctx, update.Email, user.BusinessAccountId)
		if err != nil {
			if !errors.Is(err, gorm.ErrRecordNotFound) {
				return dto.NewFailedResponse[*dto_account.User](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
			}
		}
		if existingEmail != nil && existingEmail.Id != user.Id {
			return dto.NewFailedResponse[*dto_account.User](http.StatusConflict, "this email is used by another user", nil)
		}
	}
	existing.Name = update.Name
	existing.Email = update.Email
	existing.Description = update.Description
	if err := dependencies.UnitOfWork.AccountUserRepository().Update(ctx, existing); err != nil {
		return dto.NewFailedResponse[*dto_account.User](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
	}
	d := dto_account.NewUser(*existing)
	return dto.NewSuccessResponse(&d)
}

func (UpdateProfile) APISettings() feature.APISettings {
	return feature.NewAPISettings("Update user profile", "Update logged in user's profile", types.HttpRequestTypeJSON, "PATCH", "/v1/account/users/me/profile", true, true, types.APITagUser, []feature.APIError{
		feature.NewAPIError(*exception.NewCustomException("user not found", http.StatusNotFound)),
		feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageInternalServerError, http.StatusInternalServerError)),
	})
}
