package feature_account_user

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
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

// for now only used in embedded signup
type Store struct {
	Name            string         `json:"name" val:"required" description:"name of the new user" example:"John Doe"`
	CountryCode     string         `json:"country_code" val:"required" description:"country code number"`
	PhoneNumber     string         `json:"phone_number" val:"required" description:"phone number of the user, without country code"`
	Email           string         `json:"email" description:"email address of the user"`
	Type            types.UserType `json:"type" val:"required" description:"type of the user" example:"PUBLIC"`
	Description     string         `json:"description" description:"for your reference" example:"created by account department"`
	Password        string         `json:"password" val:"required" description:"password of the user"`
	ConfirmPassword string         `json:"confirm_password" val:"required" description:"confirm password of the user"`
	// set only by embedded signup, where Meta has already proven the caller owns the phone number, not a public member
	ResumePendingPassword bool `json:"-"`
}

func (store *Store) Validate() []exception.InputException {
	store.Name = strings.TrimSpace(store.Name)
	store.Description = strings.TrimSpace(store.Description)
	store.CountryCode = strings.TrimSpace(store.CountryCode)
	store.PhoneNumber = strings.TrimSpace(store.PhoneNumber)
	store.Email = strings.TrimSpace(store.Email)
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
	if store.Email != "" && !helper.ValidateEmail(store.Email) {
		errors = append(errors, exception.NewInputException("email", "invalid email"))
	}
	if store.Type == "" {
		errors = append(errors, exception.NewInputException("type", "missing type"))
	} else if !helper.Any(types.UserTypes, func(t types.UserType) bool { return t == store.Type }) {
		errors = append(errors, exception.NewInputException("type", "invalid type"))
	}
	return errors
}

func (store Store) Handle(ctx context.Context, user *dto_account.User, dependencies *service.Dependencies) dto.Response[*dto_account.User] {
	// user is nil if coming from new embedded signup
	// if user is not nil, must be a master
	if user != nil && user.Type != types.UserTypeMaster {
		return dto.NewFailedResponse[*dto_account.User](http.StatusUnauthorized, types.ExceptionMessageUnauthorized, nil)
	}
	if errors := store.Validate(); len(errors) > 0 {
		return dto.NewInvalidInputResponse[*dto_account.User](errors)
	}
	// get existing
	existing, err := dependencies.UnitOfWork.AccountUserRepository().GetByPhoneNumber(ctx, store.CountryCode, store.PhoneNumber)
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return dto.NewFailedResponse[*dto_account.User](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
		}
	}
	// create access token and access token expiry
	accessToken, err := helper.GenerateKey(32)
	if err != nil {
		return dto.NewFailedResponse[*dto_account.User](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
	}
	accessTokenExpiry := time.Now().Add(time.Duration(cfg.Default().Site.SessionExpirySeconds) * time.Second)
	var u dto_account.User
	if existing != nil {
		u = dto_account.NewUser(*existing)
		// if store allows resume pending password and the existing user is pending password, treat it as new user
		if store.ResumePendingPassword && existing.Status == types.UserStatusPendingPassword {
			u.New = true
		}
	} else {
		// generate password hash
		passwordHash, err := helper.HashPassword(store.Password)
		if err != nil {
			return dto.NewFailedResponse[*dto_account.User](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
		}
		newUser := dao_account.User{
			Name:         store.Name,
			CountryCode:  store.CountryCode,
			PhoneNumber:  store.PhoneNumber,
			Email:        store.Email,
			Description:  store.Description,
			Type:         store.Type,
			PasswordHash: passwordHash,
			Status:       types.UserStatusPendingPassword, // new user always need to set a password
			EncryptionID: uuid.NewString(),
		}
		// Insert will do all the encryption/hashing
		if err := dependencies.UnitOfWork.AccountUserRepository().Insert(ctx, &newUser); err != nil {
			return dto.NewFailedResponse[*dto_account.User](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
		}
		u = dto_account.NewUser(newUser)
		u.New = true
	}
	// create user session only for new user, so no need to login after setting password
	if u.New {
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
			return dto.NewFailedResponse[*dto_account.User](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
		}
		// after embedded signup completed, return access token to auto login
		u.AccessToken = accessToken
		u.AccessTokenExpiry = &accessTokenExpiry
	}
	return dto.NewSuccessResponse(&u)
}

func (Store) APISettings() feature.APISettings {
	return feature.NewAPISettings("Create or update a user", "Allow user to login using password", types.HttpRequestTypeJSON, "POST", "/v1/account/users", true, false, types.APITagAccount, []feature.APIError{
		feature.NewAPIError(*exception.NewCustomException("you are not master", http.StatusBadRequest)),
		feature.NewAPIError(*exception.NewCustomException("phone number already exists", http.StatusConflict)),
		feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageInternalServerError, http.StatusInternalServerError)),
	})
}
