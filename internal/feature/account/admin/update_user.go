package feature_account_admin

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

type UpdateUser struct {
	Id             int32            `uri:"id" val:"required" description:"id of the user"`
	Name           string           `json:"name" val:"required" description:"name of the new user" example:"John Doe"`
	Email          string           `json:"email" description:"email address of the user"`
	Type           types.UserType   `json:"type" val:"required" description:"type of the user" example:"PUBLIC"`
	Status         types.UserStatus `json:"status" val:"required" description:"status of the user"`
	Description    string           `json:"description" description:"for your reference" example:"created by account department"`
	PhoneNumberIds []int32          `json:"phone_number_ids" description:"phone numbers assigned to this user"`
}

func (updateUser *UpdateUser) Validate() []exception.InputException {
	updateUser.Name = strings.TrimSpace(updateUser.Name)
	errors := []exception.InputException{}
	if updateUser.Name == "" {
		errors = append(errors, exception.NewInputException("name", "missing name"))
	}
	if updateUser.Email != "" && !helper.ValidateEmail(updateUser.Email) {
		errors = append(errors, exception.NewInputException("email", "invalid email"))
	}
	if updateUser.Type == "" {
		errors = append(errors, exception.NewInputException("type", "missing type"))
	} else if !helper.Any(types.UserTypes, func(t types.UserType) bool { return t == updateUser.Type }) {
		errors = append(errors, exception.NewInputException("type", "invalid type"))
	}
	if updateUser.Status == "" {
		errors = append(errors, exception.NewInputException("status", "missing status"))
	} else if updateUser.Status != types.UserStatusActive && updateUser.Status != types.UserStatusInactive {
		errors = append(errors, exception.NewInputException("status", "invalid status"))
	}
	return errors
}

func (updateUser UpdateUser) Handle(ctx context.Context, user *dto_account.User, dependencies *service.Dependencies) dto.Response[*dto_account.User] {
	if user == nil {
		return dto.NewFailedResponse[*dto_account.User](http.StatusForbidden, types.ExceptionMessageForbidden, nil)
	}
	if user.Type != types.UserTypeMaster {
		return dto.NewFailedResponse[*dto_account.User](http.StatusUnauthorized, types.ExceptionMessageUnauthorized, nil)
	}
	if errors := updateUser.Validate(); len(errors) > 0 {
		return dto.NewInvalidInputResponse[*dto_account.User](errors)
	}
	// get existing
	existing, err := dependencies.UnitOfWork.AccountUserRepository().GetById(ctx, updateUser.Id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return dto.NewFailedResponse[*dto_account.User](http.StatusNotFound, "user not found", nil)
		}
		return dto.NewFailedResponse[*dto_account.User](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
	}
	if existing.BusinessAccountId != user.BusinessAccountId {
		return dto.NewFailedResponse[*dto_account.User](http.StatusUnauthorized, types.ExceptionMessageUnauthorized, nil)
	}
	if existing.Type == types.UserTypeMaster && existing.Id != user.Id {
		return dto.NewFailedResponse[*dto_account.User](http.StatusUnauthorized, "you cannot edit a MASTER user", nil)
	}
	// check name exists
	nameExisting, err := dependencies.UnitOfWork.AccountUserRepository().GetByNameAndBusinessAccountId(ctx, updateUser.Name, user.BusinessAccountId)
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return dto.NewFailedResponse[*dto_account.User](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
		}
	}
	if nameExisting != nil && nameExisting.BusinessAccountId == user.BusinessAccountId && nameExisting.Id != updateUser.Id {
		return dto.NewFailedResponse[*dto_account.User](http.StatusConflict, "user with this name already exists", nil)
	}
	// check email exists
	if updateUser.Email != "" {
		emailExisting, err := dependencies.UnitOfWork.AccountUserRepository().GetByEmailAndBusinessAccountId(ctx, updateUser.Email, user.BusinessAccountId)
		if err != nil {
			if !errors.Is(err, gorm.ErrRecordNotFound) {
				return dto.NewFailedResponse[*dto_account.User](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
			}
		}
		if emailExisting != nil && emailExisting.BusinessAccountId == user.BusinessAccountId && emailExisting.Id != updateUser.Id {
			return dto.NewFailedResponse[*dto_account.User](http.StatusConflict, "user with this email already exists", nil)
		}
	}
	existing.Name = updateUser.Name
	existing.Description = updateUser.Description
	existing.Email = updateUser.Email
	existing.Status = updateUser.Status
	existing.Type = updateUser.Type
	if err := dependencies.UnitOfWork.AccountUserRepository().Update(ctx, existing); err != nil {
		return dto.NewFailedResponse[*dto_account.User](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
	}
	u := dto_account.NewUser(*existing)
	return dto.NewSuccessResponse(&u)
}

func (UpdateUser) APISettings() feature.APISettings {
	return feature.NewAPISettings("Updates a user", "Update user's profile. Only master users can access this endpoint", types.HttpRequestTypeUriJSON, "PATCH", "/v1/admin/users/:id", true, true, types.APITagAdmin, []feature.APIError{
		feature.NewAPIError(*exception.NewCustomException("you are not master", http.StatusBadRequest)),
		feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageInternalServerError, http.StatusInternalServerError)),
	})
}
