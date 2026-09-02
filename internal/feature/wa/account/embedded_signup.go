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
		dependencies.Logger.ErrorFunction(err, embeddedSignup.Data.BusinessId)
		return dto.NewFailedResponse[*dto_account.User](http.StatusBadGateway, types.ExceptionMessageBadGateway)
	}
	// Step 2 — Check the customer is telling the truth
	waba, phoneNumberDetails, response := embeddedSignup.verifyOwnership(ctx, accessToken, dependencies)
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
	// business portfolio — the tenant, storing the access token and the name from owner_business_info.
	createBusinessPortfolioResponse := (feature_wa_business_portfolio.Store{
		MetaBusinessPortfolioId: embeddedSignup.Data.BusinessId,
		Name:                    waba.OwnerBusinessInfo.Name,
		AccessToken:             accessToken,
	}).Handle(ctx, nil, &transactionDependencies)
	if !createBusinessPortfolioResponse.Success {
		return dto.NewFailedResponse[*dto_account.User](createBusinessPortfolioResponse.StatusCode, createBusinessPortfolioResponse.Message)
	}
	// Business account — the WABA under that portfolio.
	createBusinessAccountResponse := (feature_wa_business_account.Store{
		MetaBusinessProtfolioId: embeddedSignup.Data.BusinessId,
		MetaWABAId:              embeddedSignup.Data.WABAId,
	}).Handle(ctx, nil, &transactionDependencies)
	if !createBusinessAccountResponse.Success {
		return dto.NewFailedResponse[*dto_account.User](createBusinessAccountResponse.StatusCode, createBusinessAccountResponse.Message)
	}
	// Phone number — using the display number and verified name Meta reported.
	createPhoneNumberResponse := (feature_wa_phone_number.Store{
		MetaBusinessPortfolioId: embeddedSignup.Data.BusinessId,
		MetaWABAId:              embeddedSignup.Data.WABAId,
		MetaPhoneNumberId:       embeddedSignup.Data.PhoneNumberId,
		PhoneNumber:             phoneNumberDetails.DisplayPhoneNumber,
		Name:                    phoneNumberDetails.VerifiedName,
	}).Handle(ctx, nil, &transactionDependencies)
	if !createPhoneNumberResponse.Success {
		return dto.NewFailedResponse[*dto_account.User](createPhoneNumberResponse.StatusCode, createPhoneNumberResponse.Message)
	}
	// User — the login account. Before creating it you ask the database "does this portfolio already have a master?" If not, this user becomes master; otherwise they're an operator. Asking the database rather than guessing means a signup that failed halfway and got retried still produces a master.
	hasMasterUser, err := transaction.AccountUserRepository().HasMasterUser(ctx, embeddedSignup.Data.BusinessId)
	if err != nil {
		return dto.NewFailedResponse[*dto_account.User](http.StatusInternalServerError, types.ExceptionMessageInternalServerError)
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
		// Meta verified ownership above, so a signup that was abandoned before the password was set can be resumed
		ResumePendingPassword: true,
	}
	if !hasMasterUser {
		createUser.Type = types.UserTypeMaster
	}
	createUserResponse := createUser.Handle(ctx, nil, &transactionDependencies)
	if !createUserResponse.Success {
		if createUserResponse.StatusCode == http.StatusConflict {
			return dto.NewFailedResponse[*dto_account.User](http.StatusConflict, "an account already exists for this phone number, please login instead")
		}
		return dto.NewFailedResponse[*dto_account.User](createUserResponse.StatusCode, createUserResponse.Message)
	}
	// User ↔ phone number link — this is what grants the user permission to that number.
	createUserPhoneNumberResponse := (feature_wa_user_phone_number.Create{
		UserId:        createUserResponse.Data.Id,
		PhoneNumberId: createPhoneNumberResponse.Data.Id,
	}).Handle(ctx, nil, &transactionDependencies)
	if !createUserPhoneNumberResponse.Success {
		return dto.NewFailedResponse[*dto_account.User](createUserPhoneNumberResponse.StatusCode, createUserPhoneNumberResponse.Message)
	}
	// If any step fails, the deferred rollback throws away all of it. You never end up with half a tenant.
	if exception := transaction.CommitTransaction(); exception != nil {
		return dto.NewFailedResponse[*dto_account.User](http.StatusInternalServerError, types.ExceptionMessageInternalServerError)
	}
	committed = true
	// Step 4 — Commit, then talk to Meta
	// The transaction commits. Only now does activateOnMeta run, because these two calls change things on Meta's side and a database rollback can't undo them:
	embeddedSignup.activateOnMeta(ctx, accessToken, createPhoneNumberResponse.Data.Id, phoneNumberDetails.Status, dependencies)
	// Step 5 — Respond
	// The new user is returned along with a session token, so the customer lands logged in.
	return createUserResponse
}

// verifyOwnership confirms the browser supplied WABA and phone number are actually granted by the exchanged token.
// The token is scoped to the granted assets, so reading the WABA fails when it was not granted, and its
// owner_business_info proves which business portfolio the WABA really belongs to.
func (embeddedSignup EmbeddedSignup) verifyOwnership(ctx context.Context, accessToken string, dependencies *service.Dependencies) (*service.WhatsAppWABADetailsResponse, *service.WhatsAppPhoneNumberDetailsResponse, *dto.Response[*dto_account.User]) {
	forbidden := dto.NewFailedResponse[*dto_account.User](http.StatusForbidden, "the WhatsApp account is not granted to this application")
	// Read the WABA using the token. If the token doesn't cover that WABA, Meta refuses, and you return 403.
	waba, err := dependencies.Whatsapp.GetWABA(ctx, embeddedSignup.Data.WABAId, accessToken)
	if err != nil {
		return nil, nil, &forbidden
	}
	// Compare owners. The WABA response includes owner_business_info, which is Meta's own statement of which business portfolio owns it. If that doesn't match the business ID the browser sent, you return 403. This stops someone filing their WABA under somebody else's portfolio.
	if waba.OwnerBusinessInfo.ID != embeddedSignup.Data.BusinessId {
		dependencies.Logger.Warnf("embedded signup business portfolio id does not own the WABA: waba_id=%s submitted_business_id=%s owner_business_id=%s", embeddedSignup.Data.WABAId, embeddedSignup.Data.BusinessId, waba.OwnerBusinessInfo.ID)
		return nil, nil, &forbidden
	}
	// List the WABA's phone numbers and confirm the submitted phone number ID is actually one of them.
	phoneNumbers, err := dependencies.Whatsapp.GetAllPhoneNumbersByWABAId(ctx, embeddedSignup.Data.WABAId, accessToken)
	if err != nil {
		dependencies.Logger.ErrorFunction(err, embeddedSignup.Data.WABAId)
		return nil, nil, &forbidden
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
		dependencies.Logger.ErrorFunction(err, embeddedSignup.Data.PhoneNumberId)
		badGateway := dto.NewFailedResponse[*dto_account.User](http.StatusBadGateway, types.ExceptionMessageBadGateway)
		return nil, nil, &badGateway
	}
	return waba, phoneNumberDetails, nil
}

// activateOnMeta subscribes the WABA to webhooks and registers the number. Failures are logged, not returned:
// the tenant already exists and a repeated signup retries both steps.
func (embeddedSignup EmbeddedSignup) activateOnMeta(ctx context.Context, accessToken string, phoneNumberId int32, status string, dependencies *service.Dependencies) {
	// Subscribe to webhooks on the WABA, so inbound messages and delivery receipts start arriving.
	if err := dependencies.Whatsapp.SubscribeApp(ctx, embeddedSignup.Data.WABAId, accessToken); err != nil {
		dependencies.Logger.ErrorFunction(err, embeddedSignup.Data.WABAId)
	}
	if status == service.WhatsAppPhoneNumberStatusConnected {
		return
	}
	// Register the phone number, but only if it wasn't already CONNECTED. Registration generates a 6-digit two-step verification PIN, which gets saved on the phone number row — you need that same PIN to move the number between WABAs later.
	pin, err := dependencies.Whatsapp.RegisterPhoneNumber(ctx, embeddedSignup.Data.PhoneNumberId, accessToken)
	if err != nil {
		dependencies.Logger.ErrorFunction(err, embeddedSignup.Data.PhoneNumberId)
		return
	}
	phoneNumber, err := dependencies.UnitOfWork.WAPhoneNumberRepository().Get(ctx, phoneNumberId)
	if err != nil {
		dependencies.Logger.ErrorFunction(err, phoneNumberId)
		return
	}
	phoneNumber.RegistrationPin = pin
	if err := dependencies.UnitOfWork.WAPhoneNumberRepository().Update(ctx, phoneNumber); err != nil {
		dependencies.Logger.ErrorFunction(err, phoneNumberId)
	}
	// Both failures are logged rather than returned. The customer's account already exists and works; a repeated signup will retry these steps.
}

// api endpoint only for testing
func (EmbeddedSignup) APISettings() feature.APISettings {
	return feature.NewAPISettings("Process WhatsApp embedded signup", "Create the organization, WhatsApp business account, phone number, and user from an embedded signup.", types.HttpRequestTypeJSON, "POST", "/v1/wa/account/embedded-signup", false, true, types.APITagWA, []feature.APIError{
		feature.NewAPIError(*exception.NewCustomException("phone number already exists", http.StatusBadRequest)),
		feature.NewAPIError(*exception.NewCustomException("the WhatsApp account is not granted to this application", http.StatusForbidden)),
		feature.NewAPIError(*exception.NewCustomException("an account already exists for this phone number, please login instead", http.StatusConflict)),
		feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageBadGateway, http.StatusBadGateway)),
		feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageInternalServerError, http.StatusInternalServerError)),
	})
}
