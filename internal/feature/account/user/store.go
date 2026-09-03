package feature_account_user

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

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

type Store struct {
	Name            string           `json:"name" val:"required" description:"name of the new user" example:"John Doe"`
	CountryCode     string           `json:"country_code" val:"required" description:"country code number"`
	PhoneNumber     string           `json:"phone_number" val:"required" description:"phone number of the user, without country code"`
	Type            types.UserType   `json:"type" val:"required" description:"type of the user" example:"PUBLIC"`
	Description     string           `json:"description" description:"for your reference" example:"created by account department"`
	Password        string           `json:"password" val:"required" description:"password of the user"`
	ConfirmPassword string           `json:"confirm_password" val:"required" description:"confirm password of the user"`
	Status          types.UserStatus `json:"status" val:"required" description:"status of the user"`
	// set only by embedded signup, where Meta has already proven the caller owns the phone number, not a public member
	ResumePendingPassword bool `json:"-"`
}

func (store *Store) Validate() []exception.InputException {
	store.Name = strings.TrimSpace(store.Name)
	store.Description = strings.TrimSpace(store.Description)
	store.CountryCode = strings.TrimSpace(store.CountryCode)
	store.PhoneNumber = strings.TrimSpace(store.PhoneNumber)
	// remove any space or + or - from phone number
	store.PhoneNumber = strings.ReplaceAll(store.PhoneNumber, "+", "")
	store.PhoneNumber = strings.ReplaceAll(store.PhoneNumber, " ", "")
	store.PhoneNumber = strings.ReplaceAll(store.PhoneNumber, "-", "")
	store.Password = strings.TrimSpace(store.Password)
	store.ConfirmPassword = strings.TrimSpace(store.ConfirmPassword)
	errors := []exception.InputException{}
	if store.Name == "" {
		errors = append(errors, exception.NewInputException("name", "missing name"))
	}
	if store.CountryCode == "" {
		errors = append(errors, exception.NewInputException("country_code", "missing country code"))
	}
	if store.PhoneNumber == "" {
		errors = append(errors, exception.NewInputException("phone_number", "missing phone number"))
	}
	if store.Type == "" {
		errors = append(errors, exception.NewInputException("type", "missing type"))
	} else if !helper.Any(types.UserTypes, func(t types.UserType) bool { return t == store.Type }) {
		errors = append(errors, exception.NewInputException("type", "invalid type"))
	}
	if store.Status == "" {
		errors = append(errors, exception.NewInputException("status", "missing status"))
	} else if store.Status != types.UserStatusActive && store.Status != types.UserStatusPendingPassword && store.Status != types.UserStatusInactive {
		errors = append(errors, exception.NewInputException("status", "invalid status"))
	}
	return errors
}

func (store Store) Handle(ctx context.Context, user *dto_account.User, dependencies *service.Dependencies) dto.Response[*dto_account.User] {
	if user != nil && user.Type != types.UserTypeMaster {
		return dto.NewFailedResponse[*dto_account.User](http.StatusBadRequest, "you are not master")
	}
	if errors := store.Validate(); len(errors) > 0 {
		return dto.NewInvalidInputResponse[*dto_account.User](errors)
	}
	// get existing
	existing, err := dependencies.UnitOfWork.AccountUserRepository().GetByPhoneNumber(ctx, store.CountryCode, store.PhoneNumber)
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return dto.NewFailedResponse[*dto_account.User](http.StatusInternalServerError, types.ExceptionMessageInternalServerError)
		}
	}
	// create access token and access token expiry
	accessToken, err := helper.GenerateKey(32)
	if err != nil {
		dependencies.Logger.ErrorFunction(err, "failed to generate access token")
		return dto.NewFailedResponse[*dto_account.User](http.StatusInternalServerError, types.ExceptionMessageInternalServerError)
	}
	accessTokenExpiry := time.Now().Add(time.Duration(cfg.Default().Site.SessionExpirySeconds) * time.Second)
	var u dto_account.User
	if existing != nil {
		// a user who never set a password has no credential to bypass, so signup may hand back a session to finish onboarding
		if !store.ResumePendingPassword || existing.Status != types.UserStatusPendingPassword {
			return dto.NewFailedResponse[*dto_account.User](http.StatusConflict, "phone number already exists, please login instead")
		}
		u = dto_account.NewUser(*existing)
	} else {
		// generate password hash
		passwordHash, err := helper.HashPassword(store.Password)
		if err != nil {
			dependencies.Logger.ErrorFunction(err, store.PhoneNumber)
			return dto.NewFailedResponse[*dto_account.User](http.StatusInternalServerError, types.ExceptionMessageInternalServerError)
		}
		newUser := dao_account.User{
			Name:         store.Name,
			CountryCode:  store.CountryCode,
			PhoneNumber:  store.PhoneNumber,
			Description:  store.Description,
			Type:         store.Type,
			PasswordHash: passwordHash,
			Status:       store.Status,
		}
		if err := dependencies.UnitOfWork.AccountUserRepository().Insert(ctx, &newUser); err != nil {
			return dto.NewFailedResponse[*dto_account.User](http.StatusInternalServerError, types.ExceptionMessageInternalServerError)
		}
		u = dto_account.NewUser(newUser)
	}
	// create user session
	session := dao_account.Session{
		UserId:            u.Id,
		AccessTokenHashed: helper.HashSHA256Hex(accessToken),
		ExpiresAt:         accessTokenExpiry,
		LastUsedAt:        time.Now(),
	}
	if ip := helper.GetClientIP(ctx); ip != nil {
		session.IP = *ip
	}
	if userAgent := helper.GetUserAgent(ctx); userAgent != nil {
		session.UserAgent = *userAgent
	}
	if err := dependencies.UnitOfWork.AccountSessionRepository().Insert(ctx, &session); err != nil {
		return dto.NewFailedResponse[*dto_account.User](http.StatusInternalServerError, types.ExceptionMessageInternalServerError)
	}
	u.AccessToken = accessToken
	u.AccessTokenExpiry = &accessTokenExpiry
	return dto.NewSuccessResponse(&u)
}

func (Store) APISettings() feature.APISettings {
	return feature.NewAPISettings("Create or update a user", "Allow user to login using password", types.HttpRequestTypeJSON, "POST", "/v1/account/users", true, false, types.APITagAccount, []feature.APIError{
		feature.NewAPIError(*exception.NewCustomException("you are not master", http.StatusBadRequest)),
		feature.NewAPIError(*exception.NewCustomException("phone number already exists", http.StatusConflict)),
		feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageInternalServerError, http.StatusInternalServerError)),
	})
}
