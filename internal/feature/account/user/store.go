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

// only used in embedded signup
type Store struct {
	Name               string
	DisplayPhoneNumber string
	BusinessAccountId  int32
	PhoneNumberId      int32
}

func (store *Store) Validate() []exception.InputException {
	store.Name = strings.TrimSpace(store.Name)
	store.DisplayPhoneNumber = strings.TrimSpace(store.DisplayPhoneNumber)
	errors := []exception.InputException{}
	if store.Name == "" {
		errors = append(errors, exception.NewInputException("name", "missing name"))
	}
	if store.DisplayPhoneNumber == "" {
		errors = append(errors, exception.NewInputException("display_phone_number", "missing display phone number"))
	}
	if store.BusinessAccountId <= 0 {
		errors = append(errors, exception.NewInputException("business_account_id", "missing business account id"))
	}
	if store.PhoneNumberId <= 0 {
		errors = append(errors, exception.NewInputException("phone_number_id", "missing phone number id"))
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
	split := strings.Split(store.DisplayPhoneNumber, " ")
	countryCode := strings.ReplaceAll(split[0], "+", "")
	countryCode = strings.TrimSpace(countryCode)
	phoneNumber := strings.TrimPrefix(store.DisplayPhoneNumber, split[0])
	phoneNumber = strings.ReplaceAll(phoneNumber, "-", "")
	phoneNumber = strings.ReplaceAll(phoneNumber, " ", "")
	phoneNumber = strings.TrimSpace(phoneNumber)
	// get existing
	existing, err := dependencies.UnitOfWork.AccountUserRepository().GetByPhoneNumber(ctx, countryCode, phoneNumber)
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return dto.NewFailedResponse[*dto_account.User](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
		}
	}
	if existing != nil && existing.BusinessAccountId != store.BusinessAccountId {
		return dto.NewFailedResponse[*dto_account.User](http.StatusConflict, "this phone number is alrady used in another WhatsApp Business Account", nil)
	}
	// if from a unlogged in page to embedded signupd
	if user == nil {
		var u dto_account.User
		var createSession bool
		// if has existing user, pull it
		if existing != nil {
			u = dto_account.NewUser(*existing)
			// if has existing and status is pending password, create session and redirect to change password page
			// if has existing and status is not pending password, do not create session, redirect to login
			if u.Status == types.UserStatusPendingPassword {
				createSession = true
			}
		} else { // create new user
			hasMasterUser, err := dependencies.UnitOfWork.AccountUserRepository().HasMasterUserInBusinessAccount(ctx, store.BusinessAccountId)
			if err != nil {
				return dto.NewFailedResponse[*dto_account.User](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
			}
			// generate password hash
			fakePassword := uuid.NewString()
			passwordHash, err := helper.HashPassword(fakePassword)
			if err != nil {
				return dto.NewFailedResponse[*dto_account.User](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
			}
			newUser := dao_account.User{
				Name:              store.Name,
				CountryCode:       countryCode,
				PhoneNumber:       phoneNumber,
				Description:       "created from embedded signup",
				PasswordHash:      passwordHash,
				EncryptionID:      uuid.NewString(),
				BusinessAccountId: store.BusinessAccountId,
				Status:            types.UserStatusPendingPassword,
			}
			// if has no existing master user, this new user will be the master; otherwise it's an operator
			if !hasMasterUser {
				newUser.Type = types.UserTypeMaster
			} else {
				newUser.Type = types.UserTypeOperator
			}
			if err := dependencies.UnitOfWork.AccountUserRepository().Insert(ctx, &newUser); err != nil {
				return dto.NewFailedResponse[*dto_account.User](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
			}
			u = dto_account.NewUser(newUser)
			u.New = true
			createSession = true
		}
		// check this phone number is assigned to this user
		userPhoneNumber, err := dependencies.UnitOfWork.AccountUserPhoneNumberRepository().GetByUserIdAndPhoneNumberId(ctx, u.Id, store.PhoneNumberId)
		if err != nil {
			if !errors.Is(err, gorm.ErrRecordNotFound) {
				return dto.NewFailedResponse[*dto_account.User](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
			}
		}
		// if not assigned, assign now
		if userPhoneNumber == nil {
			userPhoneNumber := dao_account.UserPhoneNumber{
				UserId:        u.Id,
				PhoneNumberId: store.PhoneNumberId,
			}
			if err := dependencies.UnitOfWork.AccountUserPhoneNumberRepository().Insert(ctx, &userPhoneNumber); err != nil {
				return dto.NewFailedResponse[*dto_account.User](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
			}
		}
		// create user session only for new user, so no need to login after setting password
		if createSession {
			// create access token and access token expiry
			accessToken, err := helper.GenerateKey(32)
			if err != nil {
				return dto.NewFailedResponse[*dto_account.User](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
			}
			accessTokenExpiry := time.Now().Add(time.Duration(cfg.Default().Site.SessionExpirySeconds) * time.Second)
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
	} else { // if from a logged in page
		// return nil, to indicate next step is to assign this phone number to a user
		return dto.NewSuccessResponse[*dto_account.User](nil)
	}
}

func (Store) APISettings() feature.APISettings {
	return feature.NewAPISettings("Create or update a user", "Allow user to login using password", types.HttpRequestTypeJSON, "POST", "/v1/account/users", true, false, types.APITagAccount, []feature.APIError{
		feature.NewAPIError(*exception.NewCustomException("you are not master", http.StatusBadRequest)),
		feature.NewAPIError(*exception.NewCustomException("phone number already exists", http.StatusConflict)),
		feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageInternalServerError, http.StatusInternalServerError)),
	})
}
