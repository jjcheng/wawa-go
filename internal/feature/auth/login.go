package feature_auth

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

type Login struct {
	PhoneNumber string `json:"phone_number" val:"required" description:"user's full phone number" example:"6590909090"`
	Password    string `json:"password" val:"required" description:"user's password"`
}

func (login *Login) Validate() []exception.InputException {
	login.PhoneNumber = strings.TrimSpace(login.PhoneNumber)
	login.Password = strings.TrimSpace(login.Password)
	errors := []exception.InputException{}
	if login.PhoneNumber == "" {
		errors = append(errors, exception.NewInputException("phone_number", "missing phone number"))
	} else if !helper.IsValidPhoneNumber(login.PhoneNumber) {
		errors = append(errors, exception.NewInputException("phone_number", "invalid phone number"))
	}
	if login.Password == "" {
		errors = append(errors, exception.NewInputException("password", "missing password"))
	} else if passwordError := helper.ValidatePassword(login.Password); passwordError != nil {
		errors = append(errors, exception.NewInputException("password", passwordError.Error()))
	}
	return errors
}

func (login Login) Handle(ctx context.Context, _ *dto_account.User, dependencies *service.Dependencies) dto.Response[*dto_account.User] {
	if errors := login.Validate(); len(errors) > 0 {
		return dto.NewInvalidInputResponse[*dto_account.User](errors)
	}
	user, ex := dependencies.UnitOfWork.AccountUserRepository().GetByPhoneNumber(ctx, login.PhoneNumber)
	if ex != nil {
		if ex.StatusCode == http.StatusNotFound {
			return dto.NewFailedResponse[*dto_account.User](http.StatusUnauthorized, "invalid phone number or password")
		}
		return dto.NewFailedResponse[*dto_account.User](ex.StatusCode, ex.Message)
	}
	if user.Status == types.UserStatusInactive {
		return dto.NewFailedResponse[*dto_account.User](http.StatusUnauthorized, "user is inactive")
	}
	if !helper.VerifyPassword(login.Password, user.PasswordHash) {
		return dto.NewFailedResponse[*dto_account.User](http.StatusUnauthorized, "invalid phone number or password")
	}
	result := dto_account.NewUser(*user)
	return dto.NewSuccessResponse(&result)
}

func (Login) APISettings() feature.APISettings {
	return feature.NewAPISettings(
		"User login",
		"Authenticates a user with their phone number and password.",
		types.HttpRequestTypeJSON,
		http.MethodPost,
		"/auth/v1/login",
		false,
		true,
		types.APITagAuth,
		[]feature.APIError{
			feature.NewAPIError(*exception.NewCustomException("invalid phone number or password", http.StatusUnauthorized)),
			feature.NewAPIError(*exception.NewCustomException("user is inactive", http.StatusUnauthorized)),
		},
	)
}
