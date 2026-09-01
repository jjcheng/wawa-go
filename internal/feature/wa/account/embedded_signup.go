package feature_wa_account

import (
	"context"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/jjcheng/wawa-go/internal/dto"
	dto_account "github.com/jjcheng/wawa-go/internal/dto/account"
	"github.com/jjcheng/wawa-go/internal/exception"
	"github.com/jjcheng/wawa-go/internal/feature"
	feature_account_user "github.com/jjcheng/wawa-go/internal/feature/account/user"
	feature_wa_business_account "github.com/jjcheng/wawa-go/internal/feature/wa/business_account"
	feature_wa_business_portfolio "github.com/jjcheng/wawa-go/internal/feature/wa/business_portfolio"
	feature_wa_phone_number "github.com/jjcheng/wawa-go/internal/feature/wa/phone_number"
	feature_wa_user_phone_number "github.com/jjcheng/wawa-go/internal/feature/wa/user_phone_number"
	"github.com/jjcheng/wawa-go/internal/service"
	"github.com/jjcheng/wawa-go/internal/types"
)

type EmbeddedSignup struct {
	Type              string             `json:"type" val:"required" description:"type of the sign up" example:"WA_EMBEDDED_SIGNUP"`
	Event             string             `json:"event" val:"required" description:"event of the embedded sign up" example:"FINISH"`
	Data              EmbeddedSignupData `json:"data" val:"required" description:"WhatsApp embedded signup data"`
	AuthorizationCode string             `json:"authorization_code" val:"required" description:"to get the Business Integration System User Access Token"`
}

type EmbeddedSignupData struct {
	PhoneNumberId string `json:"phone_number_id" val:"required" description:"customer business phone number ID"`
	WABAId        string `json:"waba_id" val:"required" description:"customer WhatsApp Business Account ID"`
	BusinessId    string `json:"business_id" val:"required" description:"customer business portfolio ID"`
}

func (embeddedSignup *EmbeddedSignup) Validate() []exception.InputException {
	var errors []exception.InputException
	embeddedSignup.Event = strings.TrimSpace(embeddedSignup.Event)
	embeddedSignup.Type = strings.TrimSpace(embeddedSignup.Type)
	embeddedSignup.Data.PhoneNumberId = strings.TrimSpace(embeddedSignup.Data.PhoneNumberId)
	embeddedSignup.Data.WABAId = strings.TrimSpace(embeddedSignup.Data.WABAId)
	embeddedSignup.Data.BusinessId = strings.TrimSpace(embeddedSignup.Data.BusinessId)
	embeddedSignup.AuthorizationCode = strings.TrimSpace(embeddedSignup.AuthorizationCode)
	if embeddedSignup.Type != "WA_EMBEDDED_SIGNUP" {
		errors = append(errors, exception.NewInputException("type", "invalid event"))
	}
	if embeddedSignup.Event != "FINISH" {
		errors = append(errors, exception.NewInputException("event", "event is not FINISH"))
	}
	if embeddedSignup.Data.PhoneNumberId == "" {
		errors = append(errors, exception.NewInputException("data.phone_number_id", "missing phone number id"))
	}
	if embeddedSignup.Data.WABAId == "" {
		errors = append(errors, exception.NewInputException("data.waba_id", "missing WABA id"))
	}
	if embeddedSignup.Data.BusinessId == "" {
		errors = append(errors, exception.NewInputException("data.business_id", "missing business id"))
	}
	if embeddedSignup.AuthorizationCode == "" {
		errors = append(errors, exception.NewInputException("authorization_code", "missing authorization_code"))
	}
	return errors
}

func (embeddedSignup EmbeddedSignup) Handle(ctx context.Context, user *dto_account.User, dependencies *service.Dependencies) dto.Response[*dto_account.User] {
	if errors := embeddedSignup.Validate(); len(errors) > 0 {
		return dto.NewInvalidInputResponse[*dto_account.User](errors)
	}
	// exchange authorization code for business access token
	accessToken, err := dependencies.Whatsapp.ExchangeAccessToken(ctx, embeddedSignup.AuthorizationCode)
	if err != nil {
		return dto.NewFailedResponse[*dto_account.User](http.StatusBadGateway, err.Error())
	}
	// create/update business portfolio
	createBusinessPortfolioResponse := (feature_wa_business_portfolio.Store{
		MetaBusinessPortfolioId: embeddedSignup.Data.BusinessId,
		AccessToken:             accessToken,
	}).Handle(ctx, nil, dependencies)
	if !createBusinessPortfolioResponse.Success {
		return dto.NewFailedResponse[*dto_account.User](createBusinessPortfolioResponse.StatusCode, createBusinessPortfolioResponse.Message)
	}
	// create/update business account (WABA)
	createBusinessAccountResponse := (feature_wa_business_account.Store{
		MetaBusinessProtfolioId: embeddedSignup.Data.BusinessId,
		MetaWABAId:              embeddedSignup.Data.WABAId,
	}).Handle(ctx, nil, dependencies)
	if !createBusinessAccountResponse.Success {
		return dto.NewFailedResponse[*dto_account.User](createBusinessAccountResponse.StatusCode, createBusinessAccountResponse.Message)
	}
	// create/update phone number
	createPhoneNumberResponse := (feature_wa_phone_number.Store{
		MetaBusinessPortfolioId: embeddedSignup.Data.BusinessId,
		MetaWABAId:              embeddedSignup.Data.WABAId,
		MetaPhoneNumberId:       embeddedSignup.Data.PhoneNumberId,
	}).Handle(ctx, nil, dependencies)
	if !createPhoneNumberResponse.Success {
		return dto.NewFailedResponse[*dto_account.User](createPhoneNumberResponse.StatusCode, createPhoneNumberResponse.Message)
	}
	// create user
	tmpPassword := uuid.NewString()
	createUser := feature_account_user.Store{
		Name:            createPhoneNumberResponse.Data.Name,
		PhoneNumber:     createPhoneNumberResponse.Data.PhoneNumber,
		Type:            types.UserTypeOperator,
		Description:     "created by WhatsApp embedded signup",
		Password:        tmpPassword,
		ConfirmPassword: tmpPassword,
		Status:          types.UserStatusPendingPassword,
	}
	if createBusinessPortfolioResponse.Data.New {
		createUser.Type = types.UserTypeMaster
	}
	createUserResponse := createUser.Handle(ctx, nil, dependencies)
	if !createUserResponse.Success {
		return dto.NewFailedResponse[*dto_account.User](createUserResponse.StatusCode, createUserResponse.Message)
	}
	// create user_phone_number so user can login using password
	createUserPhoneNumberResponse := (feature_wa_user_phone_number.Create{
		UserId:        createUserResponse.Data.Id,
		PhoneNumberId: createPhoneNumberResponse.Data.Id,
	}).Handle(ctx, nil, dependencies)
	if !createUserPhoneNumberResponse.Success {
		return dto.NewFailedResponse[*dto_account.User](createUserPhoneNumberResponse.StatusCode, createUserPhoneNumberResponse.Message)
	}
	return createUserResponse
}

// api endpoint only for testing
func (EmbeddedSignup) APISettings() feature.APISettings {
	return feature.NewAPISettings("Process WhatsApp embedded signup", "Create the organization, WhatsApp business account, phone number, and user from an embedded signup.", types.HttpRequestTypeJSON, "POST", "/v1/wa/account/embedded-signup", false, true, types.APITagWA, []feature.APIError{
		feature.NewAPIError(*exception.NewCustomException("phone number already exists", http.StatusBadRequest)),
		feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageBadGateway, http.StatusBadGateway)),
		feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageInternalServerError, http.StatusInternalServerError)),
	})
}
