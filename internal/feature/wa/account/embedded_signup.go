package feature_wa_account

import (
	"context"
	"fmt"
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
	"github.com/jjcheng/wawa-go/internal/helper"
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
	// Step 1 — Swap the code for a token
	accessToken, err := dependencies.Whatsapp.ExchangeAccessToken(ctx, embeddedSignup.AuthorizationCode)
	if err != nil {
		return dto.NewFailedResponse[*dto_account.User](http.StatusBadGateway, err.Error())
	}
	// Step 2 — Check the customer is telling the truth
	wabaDetails, phoneNumberDetails, response := embeddedSignup.verifyOwnership(ctx, accessToken, dependencies)
	if response != nil {
		return *response
	}
	// Step 3 — Write everything to the database, all or nothing
	// A transaction opens. Five rows get created or updated, in dependency order:
	transaction := dependencies.UnitOfWork.BeginTransaction()
	transactionDependencies := *dependencies
	transactionDependencies.UnitOfWork = transaction
	committed := false
	defer func() {
		if !committed {
			transaction.Rollback()
		}
	}()
	// business portfolio — store the access token and the name from owner_business_info
	storeBusinessPortfolioResponse := (feature_wa_business_portfolio.Store{
		MetaBusinessPortfolioId: embeddedSignup.Data.BusinessId,
		Name:                    wabaDetails.OwnerBusinessInfo.Name,
		AccessToken:             accessToken,
	}).Handle(ctx, nil, &transactionDependencies)
	if !storeBusinessPortfolioResponse.Success {
		return dto.NewFailedResponse[*dto_account.User](storeBusinessPortfolioResponse.StatusCode, storeBusinessPortfolioResponse.Message)
	}
	// business account — the WABA under that portfolio
	storeBusinessAccountResponse := (feature_wa_business_account.Store{
		BusinessPortfolio: storeBusinessPortfolioResponse.Data,
		WABADetails:       wabaDetails,
	}).Handle(ctx, nil, &transactionDependencies)
	if !storeBusinessAccountResponse.Success {
		return dto.NewFailedResponse[*dto_account.User](storeBusinessAccountResponse.StatusCode, storeBusinessAccountResponse.Message)
	}
	// user — the login account. Before creating it you ask the database "does this portfolio already have a master?" If not, this user becomes master; otherwise they're an operator. Asking the database rather than guessing means a signup that failed halfway and got retried still produces a master.
	hasMasterUser, err := transaction.AccountUserRepository().HasMasterUser(ctx, storeBusinessAccountResponse.Data.Id)
	if err != nil {
		return dto.NewFailedResponse[*dto_account.User](http.StatusInternalServerError, types.ExceptionMessageInternalServerError)
	}
	// create user
	tmpPassword := uuid.NewString()
	// extract country code and phone number
	split := strings.Split(phoneNumberDetails.DisplayPhoneNumber, " ")
	countryCode := strings.ReplaceAll(split[0], "+", "")
	countryCode = strings.TrimSpace(countryCode)
	phoneNumber := strings.TrimPrefix(phoneNumberDetails.DisplayPhoneNumber, split[0])
	phoneNumber = strings.ReplaceAll(phoneNumber, "-", "")
	phoneNumber = strings.ReplaceAll(phoneNumber, " ", "")
	phoneNumber = strings.TrimSpace(phoneNumber)
	storeUser := feature_account_user.Store{
		Name:            phoneNumberDetails.VerifiedName,
		CountryCode:     countryCode,
		PhoneNumber:     phoneNumber,
		Type:            types.UserTypeOperator,
		Description:     "created by WhatsApp embedded signup",
		Password:        tmpPassword,
		ConfirmPassword: tmpPassword,
		// Meta verified ownership above, so a signup that was abandoned before the password was set can be resumed
		ResumePendingPassword: true,
	}
	if !hasMasterUser {
		storeUser.Type = types.UserTypeMaster
	}
	storeUserResponse := storeUser.Handle(ctx, nil, &transactionDependencies)
	if !storeUserResponse.Success {
		return storeUserResponse
	}
	// phone number — using the display number and verified name Meta reported
	storePhoneNumberResponse := (feature_wa_phone_number.Store{
		BusinessAccountId:  storeBusinessAccountResponse.Data.Id,
		MetaPhoneNumberId:  embeddedSignup.Data.PhoneNumberId,
		DisplayPhoneNumber: phoneNumberDetails.DisplayPhoneNumber,
		Name:               phoneNumberDetails.VerifiedName,
		UserId:             storeUserResponse.Data.Id,
	}).Handle(ctx, nil, &transactionDependencies)
	if !storePhoneNumberResponse.Success {
		return dto.NewFailedResponse[*dto_account.User](storePhoneNumberResponse.StatusCode, storePhoneNumberResponse.Message)
	}
	// If any step fails, the deferred rollback throws away all of it. You never end up with half a tenant.
	if exception := transaction.CommitTransaction(); exception != nil {
		return dto.NewFailedResponse[*dto_account.User](http.StatusInternalServerError, types.ExceptionMessageInternalServerError)
	}
	committed = true
	// Step 4 — Commit, then subscribe the app and activate the phone number
	// The transaction commits. Only now does activateOnMeta run, because these two calls change things on Meta's side and a database rollback can't undo them:
	if err := embeddedSignup.activateOnMeta(ctx, accessToken, storePhoneNumberResponse.Data.Id, phoneNumberDetails.Status, dependencies); err != nil {
		storeUserResponse.Data.WAActivationError = fmt.Sprintf("Your account was created, but your WhatsApp phone number could not be activated by Meta at the moment. Please try again later. Error from Meta: %v", err)
		return storeUserResponse
	}
	storeUserResponse.Data.WAActivated = true
	// Step 5 — Respond
	// The new user is returned along with a session token, so the customer lands logged in
	return storeUserResponse
}

// verifyOwnership confirms the browser supplied WABA and phone number are actually granted by the exchanged token.
// The token is scoped to the granted assets, so reading the WABA fails when it was not granted, and its
// owner_business_info proves which business portfolio the WABA really belongs to.
func (embeddedSignup EmbeddedSignup) verifyOwnership(ctx context.Context, accessToken string, dependencies *service.Dependencies) (*service.WhatsAppWABADetailsResponse, *service.WhatsAppPhoneNumberDetailsResponse, *dto.Response[*dto_account.User]) {
	forbidden := dto.NewFailedResponse[*dto_account.User](http.StatusForbidden, "the WhatsApp account is not granted to this application")
	// Read the WABA using the token. If the token doesn't cover that WABA, Meta refuses, and you return 403.
	wabaDetails, err := dependencies.Whatsapp.GetWABA(ctx, embeddedSignup.Data.WABAId, accessToken)
	if err != nil {
		failed := dto.NewFailedResponse[*dto_account.User](http.StatusBadGateway, err.Error())
		return nil, nil, &failed
	}
	// Compare owners. The WABA response includes owner_business_info, which is Meta's own statement of which business portfolio owns it. If that doesn't match the business ID the browser sent, you return 403. This stops someone filing their WABA under somebody else's portfolio.
	if wabaDetails.OwnerBusinessInfo.ID != embeddedSignup.Data.BusinessId {
		dependencies.Logger.Warnf("embedded signup business portfolio id does not own the WABA: waba_id=%s submitted_business_id=%s owner_business_id=%s", embeddedSignup.Data.WABAId, embeddedSignup.Data.BusinessId, wabaDetails.OwnerBusinessInfo.ID)
		return nil, nil, &forbidden
	}
	// List the WABA's phone numbers and confirm the submitted phone number ID is actually one of them.
	phoneNumbers, err := dependencies.Whatsapp.ListPhoneNumbers(ctx, embeddedSignup.Data.WABAId, accessToken)
	if err != nil {
		failed := dto.NewFailedResponse[*dto_account.User](http.StatusBadGateway, err.Error())
		return nil, nil, &failed
	}
	if !helper.Any(phoneNumbers, func(phoneNumber service.WhatsAppPhoneNumberDetailsResponse) bool {
		return phoneNumber.ID == embeddedSignup.Data.PhoneNumberId
	}) {
		dependencies.Logger.Warnf("embedded signup phone number id not granted: waba_id=%s phone_number_id=%s", embeddedSignup.Data.WABAId, embeddedSignup.Data.PhoneNumberId)
		return nil, nil, &forbidden
	}
	// Read that phone number on its own, because the list doesn't include the status field and you need to know whether it's already live.
	phoneNumberDetails, err := dependencies.Whatsapp.GetPhoneNumber(ctx, embeddedSignup.Data.PhoneNumberId, accessToken)
	if err != nil {
		badGateway := dto.NewFailedResponse[*dto_account.User](http.StatusBadGateway, err.Error())
		return nil, nil, &badGateway
	}
	return wabaDetails, phoneNumberDetails, nil
}

func (embeddedSignup EmbeddedSignup) activateOnMeta(ctx context.Context, accessToken string, phoneNumberId int32, status string, dependencies *service.Dependencies) error {
	// Subscribe to webhooks on the WABA, so inbound messages and delivery receipts start arriving.
	if err := dependencies.Whatsapp.SubscribeApp(ctx, embeddedSignup.Data.WABAId, accessToken); err != nil {
		return err
	}
	if status == service.WhatsAppPhoneNumberStatusConnected {
		return nil
	}
	// Register the phone number, but only if it wasn't already CONNECTED. Registration generates a 6-digit two-step verification PIN, which gets saved on the phone number row — you need that same PIN to move the number between WABAs later.
	pin, err := dependencies.Whatsapp.RegisterPhoneNumber(ctx, embeddedSignup.Data.PhoneNumberId, accessToken)
	if err != nil {
		return err
	}
	phoneNumber, err := dependencies.UnitOfWork.WAPhoneNumberRepository().Get(ctx, phoneNumberId)
	if err != nil {
		return err
	}
	phoneNumber.RegistrationPin = pin
	phoneNumber.Status = types.WAPhoneNumberStatusConnected
	if err := dependencies.UnitOfWork.WAPhoneNumberRepository().Update(ctx, phoneNumber); err != nil {
		return err
	}
	return nil
}

// api endpoint only for testing
func (EmbeddedSignup) APISettings() feature.APISettings {
	return feature.NewAPISettings("Complete WhatsApp embedded signup", "Create the Meta business portfolio, WhatsApp business account, phone number, user and activate with Meta from an embedded signup.", types.HttpRequestTypeJSON, "POST", "/v1/wa/account/embedded-signup", false, true, types.APITagWA, []feature.APIError{
		feature.NewAPIError(*exception.NewCustomException("phone number already exists", http.StatusBadRequest)),
		feature.NewAPIError(*exception.NewCustomException("the WhatsApp account is not granted to this application", http.StatusForbidden)),
		feature.NewAPIError(*exception.NewCustomException("an account already exists for this phone number, please login instead", http.StatusConflict)),
		feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageBadGateway, http.StatusBadGateway)),
		feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageInternalServerError, http.StatusInternalServerError)),
	})
}
