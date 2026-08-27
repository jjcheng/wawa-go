package feature_auth

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jjcheng/wawa-go/internal/cfg"
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
	user, err := dependencies.UnitOfWork.AccountUserRepository().GetByPhoneNumber(ctx, login.PhoneNumber)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return dto.NewFailedResponse[*dto_account.User](http.StatusNotFound, "invalid phone number or password")
		}
		return dto.NewFailedResponse[*dto_account.User](http.StatusInternalServerError, types.ExceptionMessageInternalServerError)
	}
	if !helper.VerifyPassword(login.Password, user.PasswordHash) {
		return dto.NewFailedResponse[*dto_account.User](http.StatusUnauthorized, "invalid phone number or password")
	}
	if user.Status == types.UserStatusInactive {
		return dto.NewFailedResponse[*dto_account.User](http.StatusUnauthorized, "user is inactive")
	}
	ginContext, ok := ctx.Value("gin").(*gin.Context)
	if !ok {
		dependencies.Logger.ErrorFunction(errors.New("ginContext not ok"))
		return dto.NewFailedResponse[*dto_account.User](http.StatusInternalServerError, types.ExceptionMessageInternalServerError)
	}
	sessionToken, err := helper.GenerateRandomString(64)
	if err != nil {
		dependencies.Logger.ErrorFunction(err, 64)
		return dto.NewFailedResponse[*dto_account.User](http.StatusInternalServerError, types.ExceptionMessageInternalServerError)
	}
	if err := dependencies.UnitOfWork.AccountSessionRepository().Insert(ctx, &dao_account.Session{
		UserId:     user.Id,
		TokenHash:  helper.HashSHA256Hex(sessionToken),
		ExpiresAt:  time.Now().Add(time.Duration(cfg.Default().Site.SessionExpirySeconds) * time.Second),
		LastUsedAt: time.Now(),
	}); err != nil {
		return dto.NewFailedResponse[*dto_account.User](http.StatusInternalServerError, types.ExceptionMessageInternalServerError)
	}
	secure := cfg.Default().Site.Environment != types.EnvironmentDevelop
	ginContext.SetSameSite(http.SameSiteLaxMode)
	ginContext.SetCookie(cfg.Default().Site.SessionCookieName, sessionToken, cfg.Default().Site.SessionExpirySeconds, "/", "", secure, true)
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
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageInternalServerError, http.StatusInternalServerError)),
		},
	)
}
