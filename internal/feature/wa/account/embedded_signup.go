package feature_wa_account

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/jjcheng/wawa-go/internal/dto"
	dto_account "github.com/jjcheng/wawa-go/internal/dto/account"
	dto_wa "github.com/jjcheng/wawa-go/internal/dto/wa"
	"github.com/jjcheng/wawa-go/internal/exception"
	"github.com/jjcheng/wawa-go/internal/feature"
	feature_account_notification "github.com/jjcheng/wawa-go/internal/feature/account/notification"
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

type EmbeddedSignupResult struct {
	User        *dto_account.User   `json:"user"`
	PhoneNumber *dto_wa.PhoneNumber `json:"phone_number"`
	WAError     string              `json:"wa_error"`
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

func (embeddedSignup EmbeddedSignup) Handle(ctx context.Context, user *dto_account.User, dependencies *service.Dependencies) dto.Response[*EmbeddedSignupResult] {
	if errors := embeddedSignup.Validate(); len(errors) > 0 {
		return dto.NewInvalidInputResponse[*EmbeddedSignupResult](errors)
	}
	// user is either nil or is a master
	if user != nil {
		if user.Type != types.UserTypeMaster {
			return dto.NewFailedResponse[*EmbeddedSignupResult](http.StatusUnauthorized, types.ExceptionMessageUnauthorized, nil)
		}
		if user.WA != nil && user.WA.BusinessAccount != nil && user.WA.BusinessAccount.WABAId != embeddedSignup.Data.WABAId {
			return dto.NewFailedResponse[*EmbeddedSignupResult](http.StatusConflict, "you can not sign up a phone number from a different WhatsApp Business Account", nil)
		}
	}
	// Step 1 — Swap the code for a token
	accessToken, err := dependencies.Whatsapp.ExchangeAccessToken(ctx, embeddedSignup.AuthorizationCode)
	if err != nil {
		return dto.NewFailedResponse[*EmbeddedSignupResult](http.StatusBadGateway, err.Error(), err)
	}
	// Step 2 — Check the customer is telling the truth
	wabaDetails, phoneNumberDetails, err := embeddedSignup.verifyOwnership(ctx, accessToken, dependencies)
	if err != nil {
		return dto.NewFailedResponse[*EmbeddedSignupResult](http.StatusBadGateway, err.Error(), err)
	}
	// Step 3 — Write everything to the database, all or nothing
	// A transaction opens. Five rows get created or updated, in dependency order:
	_transaction := dependencies.UnitOfWork.BeginTransaction()
	transactionDependencies := *dependencies
	transactionDependencies.UnitOfWork = _transaction
	committed := false
	defer func() {
		if !committed {
			transactionDependencies.UnitOfWork.Rollback()
		}
	}()
	// business portfolio — store the access token and the name from owner_business_info
	storeBusinessPortfolioResponse := (feature_wa_business_portfolio.Store{
		MetaBusinessPortfolioId: embeddedSignup.Data.BusinessId,
		Name:                    wabaDetails.OwnerBusinessInfo.Name,
		AccessToken:             accessToken,
	}).Handle(ctx, user, &transactionDependencies)
	if !storeBusinessPortfolioResponse.Success {
		return dto.NewFailedResponse[*EmbeddedSignupResult](storeBusinessPortfolioResponse.StatusCode, storeBusinessPortfolioResponse.Message, storeBusinessPortfolioResponse.Error)
	}
	// business account — the WABA under that portfolio
	storeBusinessAccountResponse := (feature_wa_business_account.Store{
		BusinessPortfolio: storeBusinessPortfolioResponse.Data,
		WABADetails:       wabaDetails,
	}).Handle(ctx, user, &transactionDependencies)
	if !storeBusinessAccountResponse.Success {
		return dto.NewFailedResponse[*EmbeddedSignupResult](storeBusinessAccountResponse.StatusCode, storeBusinessAccountResponse.Message, storeBusinessAccountResponse.Error)
	}
	// phone number — using the display number and verified name Meta reported
	storePhoneNumberResponse := (feature_wa_phone_number.Store{
		BusinessAccountId:  storeBusinessAccountResponse.Data.Id,
		MetaPhoneNumberId:  embeddedSignup.Data.PhoneNumberId,
		DisplayPhoneNumber: phoneNumberDetails.DisplayPhoneNumber,
		Name:               phoneNumberDetails.VerifiedName,
	}).Handle(ctx, nil, &transactionDependencies)
	if !storePhoneNumberResponse.Success {
		return dto.NewFailedResponse[*EmbeddedSignupResult](storePhoneNumberResponse.StatusCode, storePhoneNumberResponse.Message, storePhoneNumberResponse.Error)
	}
	// store user
	storeUser := feature_account_user.Store{
		Name:               phoneNumberDetails.VerifiedName,
		DisplayPhoneNumber: phoneNumberDetails.DisplayPhoneNumber,
		BusinessAccountId:  storeBusinessAccountResponse.Data.Id,
		PhoneNumberId:      storePhoneNumberResponse.Data.Id,
	}
	storeUserResponse := storeUser.Handle(ctx, user, &transactionDependencies)
	if !storeUserResponse.Success {
		return dto.NewFailedResponse[*EmbeddedSignupResult](storeUserResponse.StatusCode, storeUserResponse.Message, storeBusinessAccountResponse.Error)
	}
	// it may return success with nil user, indicate to assign phone number to users
	finalUser := storeUserResponse.Data
	// If any step fails, the deferred rollback throws away all of it. You never end up with half a tenant.
	if err := transactionDependencies.UnitOfWork.CommitTransaction(); err != nil {
		return dto.NewFailedResponse[*EmbeddedSignupResult](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
	}
	committed = true
	// Step 4 — Commit, then subscribe the app and activate the phone number
	// The transaction commits. Only now does activateOnMeta run, because these two calls change things on Meta's side and a database rollback can't undo them:
	if err := embeddedSignup.activateOnMeta(ctx, accessToken, storePhoneNumberResponse.Data.Id, phoneNumberDetails.Status, dependencies); err != nil {
		result := EmbeddedSignupResult{
			User:        finalUser,
			PhoneNumber: storePhoneNumberResponse.Data,
			WAError:     fmt.Sprintf("Your account was created, but your WhatsApp phone number could not be activated by Meta at the moment. Please try again later. Error from Meta: %v", err.Error()),
		}
		return dto.NewSuccessResponse(&result)
	}
	result := EmbeddedSignupResult{
		User:        finalUser,
		PhoneNumber: storePhoneNumberResponse.Data,
	}
	// for new user, create todos
	if storeUserResponse.Data != nil && storeUserResponse.Data.New {
		// interact with customer
		if html, err := helper.ReadFromFile("www/todos/chat_with_customer.html"); err == nil {
			createNotification := feature_account_notification.Create{
				Category: types.NotificationCategoryPending,
				IconType: types.NotificationIconTypeChat,
				Type:     types.NotificationTypeInfo,
				Title:    "Get Started: Chat with your customers",
				Body:     *html,
				URL:      "/chats",
				ToUserId: finalUser.Id,
			}
			_ = createNotification.Handle(ctx, user, dependencies)
		}
		if storeUserResponse.Data.Type == types.UserTypeMaster {
			// create websites
			if html, err := helper.ReadFromFile("www/todos/create_websites.html"); err == nil {
				createNotification := feature_account_notification.Create{
					Category: types.NotificationCategoryPending,
					IconType: types.NotificationIconTypeWebsite,
					Type:     types.NotificationTypeInfo,
					Title:    "Get Started: Create websites",
					Body:     *html,
					URL:      "/catalogs",
					ToUserId: finalUser.Id,
				}
				_ = createNotification.Handle(ctx, user, dependencies)
			}
			// broadcast to customers
			if html, err := helper.ReadFromFile("www/todos/broadcast_to_customers.html"); err == nil {
				createNotification := feature_account_notification.Create{
					Category: types.NotificationCategoryPending,
					Type:     types.NotificationTypeInfo,
					IconType: types.NotificationIconTypeBroadcast,
					Title:    "Get Started: Broadcast to your customers",
					Body:     *html,
					URL:      "/chats",
					ToUserId: finalUser.Id,
				}
				_ = createNotification.Handle(ctx, user, dependencies)
			}
			// create templates
			if html, err := helper.ReadFromFile("www/todos/create_templates.html"); err == nil {
				createNotification := feature_account_notification.Create{
					Category: types.NotificationCategoryPending,
					Type:     types.NotificationTypeInfo,
					IconType: types.NotificationIconTypeTemplate,
					Title:    "Get Started: Create message templates",
					Body:     *html,
					URL:      "/templates",
					ToUserId: finalUser.Id,
				}
				_ = createNotification.Handle(ctx, user, dependencies)
			}
			// add customers
			if html, err := helper.ReadFromFile("www/todos/add_customers.html"); err == nil {
				createNotification := feature_account_notification.Create{
					Category: types.NotificationCategoryPending,
					Type:     types.NotificationTypeInfo,
					IconType: types.NotificationIconTypeCustomer,
					Title:    "Get Started: Add more customers",
					Body:     *html,
					URL:      "/chats",
					ToUserId: finalUser.Id,
				}
				_ = createNotification.Handle(ctx, user, dependencies)
			}
			// onbard more members
			if html, err := helper.ReadFromFile("www/todos/onboard_phone_numbers.html"); err == nil {
				createNotification := feature_account_notification.Create{
					Category: types.NotificationCategoryPending,
					IconType: types.NotificationIconTypeJoin,
					Type:     types.NotificationTypeInfo,
					Title:    "Get Started: Onboard more WhatsApp phone numbers",
					Body:     *html,
					URL:      "/users",
					ToUserId: finalUser.Id,
				}
				_ = createNotification.Handle(ctx, user, dependencies)
			}
		} else {
			// broadcast to customers
			if html, err := helper.ReadFromFile("www/todos/broadcast_to_customers.html"); err == nil {
				createNotification := feature_account_notification.Create{
					Category: types.NotificationCategoryPending,
					Type:     types.NotificationTypeInfo,
					IconType: types.NotificationIconTypeBroadcast,
					Title:    "Get Started: Broadcast to your customers",
					Body:     *html,
					URL:      "/chats",
					ToUserId: finalUser.Id,
				}
				_ = createNotification.Handle(ctx, user, dependencies)
			}
			// add customers
			if html, err := helper.ReadFromFile("www/todos/add_customers.html"); err == nil {
				createNotification := feature_account_notification.Create{
					Category: types.NotificationCategoryPending,
					Type:     types.NotificationTypeInfo,
					IconType: types.NotificationIconTypeCustomer,
					Title:    "Get Started: Add more customers",
					Body:     *html,
					URL:      "/chats",
					ToUserId: finalUser.Id,
				}
				_ = createNotification.Handle(ctx, user, dependencies)
			}
		}
	}
	return dto.NewSuccessResponse(&result)
}

// verifyOwnership confirms the browser supplied WABA and phone number are actually granted by the exchanged token.
// The token is scoped to the granted assets, so reading the WABA fails when it was not granted, and its
// owner_business_info proves which business portfolio the WABA really belongs to.
func (embeddedSignup EmbeddedSignup) verifyOwnership(ctx context.Context, accessToken string, dependencies *service.Dependencies) (*service.WhatsAppWABADetailsResponse, *service.WhatsAppPhoneNumberDetailsResponse, error) {
	// Read the WABA using the token. If the token doesn't cover that WABA, Meta refuses, and you return 403.
	wabaDetails, err := dependencies.Whatsapp.GetWABA(ctx, embeddedSignup.Data.WABAId, accessToken)
	if err != nil {
		return nil, nil, err
	}
	// Compare owners. The WABA response includes owner_business_info, which is Meta's own statement of which business portfolio owns it. If that doesn't match the business ID the browser sent, you return 403. This stops someone filing their WABA under somebody else's portfolio.
	if wabaDetails.OwnerBusinessInfo.ID != embeddedSignup.Data.BusinessId {
		return nil, nil, fmt.Errorf("embedded signup business portfolio id does not own the business account")
	}
	// List the WABA's phone numbers and confirm the submitted phone number ID is actually one of them.
	phoneNumbers, err := dependencies.Whatsapp.ListPhoneNumbers(ctx, embeddedSignup.Data.WABAId, accessToken)
	if err != nil {
		return nil, nil, err
	}
	if !helper.Any(phoneNumbers, func(phoneNumber service.WhatsAppPhoneNumberDetailsResponse) bool {
		return phoneNumber.ID == embeddedSignup.Data.PhoneNumberId
	}) {
		return nil, nil, fmt.Errorf("embedded signup phone number id does not elong to the business account")
	}
	// Read that phone number on its own, because the list doesn't include the status field and you need to know whether it's already live.
	phoneNumberDetails, err := dependencies.Whatsapp.GetPhoneNumber(ctx, embeddedSignup.Data.PhoneNumberId, accessToken)
	if err != nil {
		return nil, nil, err
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
	phoneNumber, err := dependencies.UnitOfWork.WAPhoneNumberRepository().GetById(ctx, phoneNumberId)
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
