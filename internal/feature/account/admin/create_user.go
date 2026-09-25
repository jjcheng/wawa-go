package feature_account_admin

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/google/uuid"
	dao_account "github.com/jjcheng/wawa-go/internal/dao/account"
	"github.com/jjcheng/wawa-go/internal/dto"
	dto_account "github.com/jjcheng/wawa-go/internal/dto/account"
	"github.com/jjcheng/wawa-go/internal/exception"
	"github.com/jjcheng/wawa-go/internal/feature"
	"github.com/jjcheng/wawa-go/internal/helper"
	"github.com/jjcheng/wawa-go/internal/service"
	"github.com/jjcheng/wawa-go/internal/types"
	"gorm.io/gorm"
)

type CreateUser struct {
	Name            string         `json:"name" val:"required" description:"name of the new user" example:"John Doe"`
	CountryCode     string         `json:"country_code" val:"required" description:"country code number"`
	PhoneNumber     string         `json:"phone_number" val:"required" description:"phone number of the user, without country code"`
	Email           string         `json:"email" description:"email address of the user"`
	Type            types.UserType `json:"type" val:"required" description:"type of the user" example:"PUBLIC"`
	Description     string         `json:"description" description:"for your reference" example:"created by account department"`
	Password        string         `json:"password" val:"required" description:"password of the new user"`
	ConfirmPassword string         `json:"confirm_password" val:"required" description:"confirm password of the user"`
}

func (createUser *CreateUser) Validate() []exception.InputException {
	createUser.Name = strings.TrimSpace(createUser.Name)
	createUser.CountryCode = strings.TrimSpace(createUser.CountryCode)
	createUser.PhoneNumber = strings.TrimSpace(createUser.PhoneNumber)
	// remove any space or + or - from phone number
	createUser.PhoneNumber = strings.ReplaceAll(createUser.PhoneNumber, "+", "")
	createUser.PhoneNumber = strings.ReplaceAll(createUser.PhoneNumber, " ", "")
	createUser.PhoneNumber = strings.ReplaceAll(createUser.PhoneNumber, "-", "")
	errors := []exception.InputException{}
	if createUser.Name == "" {
		errors = append(errors, exception.NewInputException("name", "missing name"))
	}
	if createUser.CountryCode == "" {
		errors = append(errors, exception.NewInputException("country_code", "missing country code"))
	}
	if createUser.PhoneNumber == "" {
		errors = append(errors, exception.NewInputException("phone_number", "missing phone number"))
	}
	if createUser.Email != "" && !helper.ValidateEmail(createUser.Email) {
		errors = append(errors, exception.NewInputException("email", "invalid email"))
	}
	if createUser.Type == "" {
		errors = append(errors, exception.NewInputException("type", "missing type"))
	} else if !helper.Any(types.UserTypes, func(t types.UserType) bool { return t == createUser.Type }) {
		errors = append(errors, exception.NewInputException("type", "invalid type"))
	}
	if strings.TrimSpace(createUser.Password) == "" {
		errors = append(errors, exception.NewInputException("password", "missing password"))
	} else if err := helper.ValidatePassword(createUser.Password); err != nil {
		errors = append(errors, exception.NewInputException("password", err.Error()))
	}
	if strings.TrimSpace(createUser.ConfirmPassword) == "" {
		errors = append(errors, exception.NewInputException("confirm_password", "missing confirm_password"))
	} else if createUser.Password != createUser.ConfirmPassword {
		errors = append(errors, exception.NewInputException("confirm_password", "password and confirm password must be same"))
	}
	return errors
}

func (createUser CreateUser) Handle(ctx context.Context, user *dto_account.User, dependencies *service.Dependencies) dto.Response[*dto_account.User] {
	if user == nil {
		return dto.NewFailedResponse[*dto_account.User](http.StatusForbidden, types.ExceptionMessageForbidden, nil)
	}
	if user.Type != types.UserTypeMaster {
		return dto.NewFailedResponse[*dto_account.User](http.StatusUnauthorized, types.ExceptionMessageUnauthorized, nil)
	}
	if errors := createUser.Validate(); len(errors) > 0 {
		return dto.NewInvalidInputResponse[*dto_account.User](errors)
	}
	// get existing
	existing, err := dependencies.UnitOfWork.AccountUserRepository().GetByPhoneNumber(ctx, createUser.CountryCode, createUser.PhoneNumber)
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return dto.NewFailedResponse[*dto_account.User](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
		}
	}
	// if has existing countryCode+phoneNumber and the user has the same businessAccountId, reject
	if existing != nil && existing.BusinessAccountId == user.BusinessAccountId {
		return dto.NewFailedResponse[*dto_account.User](http.StatusConflict, "there is an existing user with the same country code + phone number", nil)
	}
	// check email exists
	if createUser.Email != "" {
		emailExisting, err := dependencies.UnitOfWork.AccountUserRepository().GetByEmailAndBusinessAccountId(ctx, createUser.Email, user.BusinessAccountId)
		if err != nil {
			if !errors.Is(err, gorm.ErrRecordNotFound) {
				return dto.NewFailedResponse[*dto_account.User](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
			}
		}
		if emailExisting != nil {
			return dto.NewFailedResponse[*dto_account.User](http.StatusConflict, "user with this email already exists", nil)
		}
	}
	// generate password hash
	passwordHash, err := helper.HashPassword(createUser.Password)
	if err != nil {
		return dto.NewFailedResponse[*dto_account.User](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
	}
	newUser := dao_account.User{
		Name:              createUser.Name,
		CountryCode:       createUser.CountryCode,
		PhoneNumber:       createUser.PhoneNumber,
		Email:             createUser.Email,
		Description:       createUser.Description,
		Type:              createUser.Type,
		Status:            types.UserStatusActive,
		PasswordHash:      passwordHash,
		EncryptionID:      uuid.NewString(),
		BusinessAccountId: user.BusinessAccountId,
	}
	// Insert will do all the encryption/hashing
	if err := dependencies.UnitOfWork.AccountUserRepository().Insert(ctx, &newUser); err != nil {
		return dto.NewFailedResponse[*dto_account.User](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
	}
	u := dto_account.NewUser(newUser)
	return dto.NewSuccessResponse(&u)
}

func (CreateUser) APISettings() feature.APISettings {
	return feature.NewAPISettings("Master user creates a user", "Only MASTER user can use this to create user", types.HttpRequestTypeJSON, "POST", "/v1/admin/users", true, false, types.APITagAccount, []feature.APIError{
		feature.NewAPIError(*exception.NewCustomException("you are not master", http.StatusBadRequest)),
		feature.NewAPIError(*exception.NewCustomException("there is an existing user with the same country code + phone number", http.StatusConflict)),
		feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageInternalServerError, http.StatusInternalServerError)),
	})
}
