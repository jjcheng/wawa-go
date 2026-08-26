package feature_wa_webhook

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/jjcheng/wawa-go/internal/dto"
	dto_account "github.com/jjcheng/wawa-go/internal/dto/account"
	dto_wa "github.com/jjcheng/wawa-go/internal/dto/wa"
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
	Event string             `json:"event" val:"required" description:"embedded signup event name" example:"whatsapp_embedded_signup"`
	Data  EmbeddedSignupData `json:"data" val:"required" description:"WhatsApp embedded signup data"`
}

type EmbeddedSignupData struct {
	PhoneNumberId string `json:"phone_number_id" val:"required" description:"customer business phone number ID"`
	WABAId        string `json:"waba_id" val:"required" description:"customer WhatsApp Business Account ID"`
	BusinessId    string `json:"business_id" val:"required" description:"customer business portfolio ID"`
	Code          string `json:"code" val:"required" description:"exchangeable token code"`
}

func (embeddedSignup *EmbeddedSignup) Validate() []exception.InputException {
	var errors []exception.InputException
	embeddedSignup.Event = strings.TrimSpace(embeddedSignup.Event)
	embeddedSignup.Data.PhoneNumberId = strings.TrimSpace(embeddedSignup.Data.PhoneNumberId)
	embeddedSignup.Data.WABAId = strings.TrimSpace(embeddedSignup.Data.WABAId)
	embeddedSignup.Data.BusinessId = strings.TrimSpace(embeddedSignup.Data.BusinessId)
	embeddedSignup.Data.Code = strings.TrimSpace(embeddedSignup.Data.Code)
	if embeddedSignup.Event != "whatsapp_embedded_signup" {
		errors = append(errors, exception.NewInputException("event", "invalid event"))
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
	if embeddedSignup.Data.Code == "" {
		errors = append(errors, exception.NewInputException("data.code", "missing code"))
	}
	return errors
}

func (embeddedSignup EmbeddedSignup) Handle(ctx context.Context, user *dto_account.User, dependencies *service.Dependencies) dto.Response[*dto_wa.EmbeddedSignupResponse] {
	if errors := embeddedSignup.Validate(); len(errors) > 0 {
		return dto.NewInvalidInputResponse[*dto_wa.EmbeddedSignupResponse](errors)
	}
	// check phone number already exist
	exist, ex := dependencies.UnitOfWork.WAPhoneNumberRepository().CheckExists(ctx, embeddedSignup.Data.PhoneNumberId)
	if ex != nil && ex.StatusCode != http.StatusNotFound {
		return dto.NewFailedResponse[*dto_wa.EmbeddedSignupResponse](ex.StatusCode, ex.Message)
	}
	if exist {
		return dto.NewFailedResponse[*dto_wa.EmbeddedSignupResponse](http.StatusBadRequest, "phone number already exists")
	}
	// create business portfolio
	createBusinessPortfolioResponse := (feature_wa_business_portfolio.Create{
		MetaBusinessPortfolioId: embeddedSignup.Data.BusinessId,
	}).Handle(ctx, nil, dependencies)
	if !createBusinessPortfolioResponse.Success {
		return dto.NewFailedResponse[*dto_wa.EmbeddedSignupResponse](createBusinessPortfolioResponse.StatusCode, createBusinessPortfolioResponse.Message)
	}
	// create business account (WABA)
	createBusinessAccountResponse := (feature_wa_business_account.Create{
		MetaBusinessProtfolioId: embeddedSignup.Data.BusinessId,
		MetaWABAId:              embeddedSignup.Data.WABAId,
	}).Handle(ctx, nil, dependencies)
	if !createBusinessAccountResponse.Success {
		return dto.NewFailedResponse[*dto_wa.EmbeddedSignupResponse](createBusinessAccountResponse.StatusCode, createBusinessAccountResponse.Message)
	}
	// create phone number
	createPhoneNumberResponse := (feature_wa_phone_number.Create{
		MetaBusinessPortfolioId: embeddedSignup.Data.BusinessId,
		MetaWABAId:              embeddedSignup.Data.WABAId,
		MetaPhoneNumberId:       embeddedSignup.Data.PhoneNumberId,
	}).Handle(ctx, nil, dependencies)
	if !createPhoneNumberResponse.Success {
		return dto.NewFailedResponse[*dto_wa.EmbeddedSignupResponse](createPhoneNumberResponse.StatusCode, createPhoneNumberResponse.Message)
	}
	// create user
	tmpPassword := uuid.NewString()
	createUser := feature_account_user.Create{
		Name:            createPhoneNumberResponse.Data.Name,
		PhoneNumber:     createPhoneNumberResponse.Data.PhoneNumber,
		Type:            types.UserTypeStaff,
		Description:     "created by WhatsApp embedded signup",
		Password:        tmpPassword,
		ConfirmPassword: tmpPassword,
		Status:          types.UserStatusPendingPassword,
	}
	if createBusinessPortfolioResponse.Data.New {
		createUser.Type = types.UserTypeAdmin
	}
	createUserResponse := createUser.Handle(ctx, nil, dependencies)
	if !createUserResponse.Success {
		return dto.NewFailedResponse[*dto_wa.EmbeddedSignupResponse](createUserResponse.StatusCode, createUserResponse.Message)
	}
	// create user_phone_number so user can login using password
	createUserPhoneNumberResponse := (feature_wa_user_phone_number.Create{
		UserId:        createUserResponse.Data.Id,
		PhoneNumberId: createPhoneNumberResponse.Data.Id,
	}).Handle(ctx, nil, dependencies)
	if !createUserPhoneNumberResponse.Success {
		return dto.NewFailedResponse[*dto_wa.EmbeddedSignupResponse](createUserPhoneNumberResponse.StatusCode, createUserPhoneNumberResponse.Message)
	}
	// finalize with meta
	if err := dependencies.Whatsapp.RegisterPhoneNumber(ctx, embeddedSignup.Data.PhoneNumberId); err != nil {
		return dto.NewFailedResponse[*dto_wa.EmbeddedSignupResponse](http.StatusBadGateway, fmt.Sprintf("error registering phone number: %v", err))
	}
	return dto.NewSuccessResponse(&dto_wa.EmbeddedSignupResponse{
		PhoneNumber: *createPhoneNumberResponse.Data,
		User:        *createUserResponse.Data,
	})
}

// api endpoint only for testing
func (EmbeddedSignup) APISettings() feature.APISettings {
	return feature.NewAPISettings("Process WhatsApp embedded signup", "Create the organization, WhatsApp business account, phone number, and user from an embedded signup.", types.HttpRequestTypeJSON, "POST", "/wa/v1/embedded-signup", false, true, types.APITagWA, nil)
}
