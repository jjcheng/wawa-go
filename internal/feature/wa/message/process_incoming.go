package feature_wa_message

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jjcheng/wawa-go/internal/cfg"
	dao_account "github.com/jjcheng/wawa-go/internal/dao/account"
	dao_wa "github.com/jjcheng/wawa-go/internal/dao/wa"
	"github.com/jjcheng/wawa-go/internal/dto"
	dto_account "github.com/jjcheng/wawa-go/internal/dto/account"
	dto_customer "github.com/jjcheng/wawa-go/internal/dto/customer"
	dto_wa "github.com/jjcheng/wawa-go/internal/dto/wa"
	feature_account_notification "github.com/jjcheng/wawa-go/internal/feature/account/notification"
	feature_customer "github.com/jjcheng/wawa-go/internal/feature/customer"
	"github.com/jjcheng/wawa-go/internal/helper"
	"github.com/jjcheng/wawa-go/internal/service"
	"github.com/jjcheng/wawa-go/internal/types"
	"gorm.io/gorm"
)

// SMQ will auto retry 2 times if this returns error
func ProcessIncoming(ctx context.Context, incoming dto_wa.Incoming, dependencies *service.Dependencies) error {
	dependencies.Logger.Infoln("processing WA incoming")
	for _, entry := range incoming.Entry {
		for _, change := range entry.Changes {
			if change.Field == "messages" {
				var incomingValue dto_wa.IncomingValue
				if err := json.Unmarshal(change.Value, &incomingValue); err != nil {
					dependencies.Logger.ErrorFunction(err)
					return fmt.Errorf("invalid messages change value")
				}
				// process incoming messages
				for _, incomingMessage := range incomingValue.Messages {
					if err := insertIncomingMessage(ctx, dependencies, incomingMessage, incomingValue.Contacts, incomingValue.Metadata); err != nil {
						// if already stored, it's not really an error
						if strings.Contains(err.Error(), "duplicate") {
							continue
						}
						return err
					}
				}
				// process incoming message statuses
				for _, status := range incomingValue.Statuses {
					if err := insertIncomingStatus(ctx, dependencies, status); err != nil {
						return err
					}
				}
			} else if change.Field == "message_template_status_update" {
				var templateStatus dto_wa.IncomingTemplateStatusChange
				if err := json.Unmarshal(change.Value, &templateStatus); err != nil {
					dependencies.Logger.ErrorFunction(err)
					return fmt.Errorf("invalid message template status change value: %w", err)
				}
				// do not store it into db, create a notification to inform user
				key := helper.GetTemplateStatusChangeCacheKey(fmt.Sprint(templateStatus.MessageTemplateId))
				cachedValue, err := dependencies.Cache.Get(ctx, key)
				if err != nil {
					dependencies.Logger.ErrorFunction(err, key)
					return nil
				}
				userId, err := strconv.Atoi(cachedValue)
				if err != nil {
					dependencies.Logger.ErrorFunction(err, userId)
					return nil
				}
				user, err := dependencies.UnitOfWork.AccountUserRepository().GetById(ctx, int32(userId))
				if err != nil {
					dependencies.Logger.ErrorFunction(err, userId)
				}
				var title, body, url string
				var notificationType types.NotificationType
				if templateStatus.Event == "APPROVED" {
					title = fmt.Sprintf("Your template %s (%s) has been approved by Meta.", templateStatus.MessageTemplateName, templateStatus.MessageTemplateLanguage)
					body = "You can now go to customers page, select at least 1 customer and start a campaign with your new template."
					notificationType = types.NotificationTypeSuccess
					url = "/customers"
				} else {
					title = fmt.Sprintf("Your template %s (%s) has been %s by Meta.", templateStatus.Event, templateStatus.MessageTemplateName, templateStatus.MessageTemplateLanguage)
					body = fmt.Sprintf("Reason returned by Meta is: %s", templateStatus.Reason)
					notificationType = types.NotificationTypeWarning
					url = "/templates"
				}
				createNotification := feature_account_notification.Create{
					Type:  notificationType,
					Title: title,
					Body:  body,
					URL:   url,
				}
				_ = createNotification.Handle(ctx, helper.ConvertToPointer(dto_account.NewUser(*user)), dependencies)
				err = dependencies.Cache.Remove(ctx, key)
				if err != nil {
					dependencies.Logger.ErrorFunction(err, key)
				}
			}
		}
	}
	return nil
}

func insertIncomingStatus(ctx context.Context, dependencies *service.Dependencies, status dto_wa.Status) error {
	timestamp, err := strconv.ParseInt(status.Timestamp, 10, 64)
	if err != nil {
		dependencies.Logger.ErrorFunction(err, status.Timestamp)
		return fmt.Errorf("message status timestamp invalid: %s", status.Timestamp)
	}
	transaction := dependencies.UnitOfWork.BeginTransaction()
	committed := false
	defer func() {
		if !committed {
			transaction.Rollback()
		}
	}()
	messageStatus := types.WAMessageStatus(strings.ToUpper(status.Status))
	message, err := transaction.WAMessageRepository().GetByWAMessageId(ctx, status.ID)
	if err != nil {
		// it's possible that the message was stored without wa message id, use biz_opaque_callback_data to check
		if status.BizOpaqueCallbackData != "" {
			message, err = transaction.WAMessageRepository().GetByToken(ctx, status.BizOpaqueCallbackData)
		}
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return fmt.Errorf("message not found")
			}
			dependencies.Logger.ErrorFunction(err, status.ID)
			return fmt.Errorf("failed to get message from status")
		}
	}
	event := dao_wa.MessageStatus{
		WAMessageId:  status.ID,
		MessageId:    message.Id,
		Status:       messageStatus,
		Timestamp:    timestamp,
		Payload:      status.Payload,
		EncryptionID: uuid.NewString(),
	}
	// build error message
	if len(status.Errors) > 0 {
		var errors []string
		for _, err := range status.Errors {
			errors = append(errors, err.Error())
		}
		event.ErrorMessage = strings.Join(errors, "\n\n")
	}
	if err := transaction.WAMessageStatusRepository().Insert(ctx, &event); err != nil {
		dependencies.Logger.ErrorFunction(err, status.ID, message.Id, messageStatus, timestamp)
		return fmt.Errorf("failed to insert message status")
	}
	// Persist every callback for audit, but only advance the current status. Meta may
	// deliver callbacks out of order or omit delivered when a message is read directly.
	// This prevents stale sent, delivered, or failed callbacks from regressing read/played.
	if helper.CanTransitionWAMessageStatus(message.Status, messageStatus) {
		message.Status = messageStatus
		message.WAMessageId = status.ID // ensure waMessageId is there, it may be missing
		message.ErrorMessage = event.ErrorMessage
		if status.Pricing != nil {
			message.Billable = status.Pricing.Billable
			message.BillingType = status.Pricing.Type
			message.Category = status.Pricing.Category
		}
		if err := transaction.WAMessageRepository().Update(ctx, message); err != nil {
			dependencies.Logger.ErrorFunction(err, messageStatus, status.ID, event.ErrorMessage)
			return fmt.Errorf("failed to update message")
		}
	}
	// get customer
	customer, err := transaction.CustomerRepository().GetById(ctx, message.CustomerId)
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			dependencies.Logger.ErrorFunction(err, message.CustomerId)
			return fmt.Errorf("customer not found")
		}
		return fmt.Errorf("failed to get customer from message")
	}
	// update customer if needed
	var customerHasChange bool
	if status.RecipientUserID != "" && customer.MetaUserId != status.RecipientUserID {
		customer.MetaUserId = status.RecipientUserID
		customerHasChange = true
	}
	if status.RecipientID != "" && customer.WAId != status.RecipientID {
		customer.WAId = status.RecipientID
		customerHasChange = true
	}
	if customerHasChange {
		if err := transaction.CustomerRepository().Update(ctx, customer); err != nil {
			dependencies.Logger.ErrorFunction(err, customer.Id)
			return fmt.Errorf("failed update customer Meta user id or WA id")
		}
	}
	// get phone number
	phoneNumber, err := transaction.WAPhoneNumberRepository().GetById(ctx, message.PhoneNumberId)
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			dependencies.Logger.ErrorFunction(err, message.PhoneNumberId)
			return fmt.Errorf("phone number not found")
		}
		return fmt.Errorf("failed to get phone number")
	}
	// now commit the transaction
	if err := transaction.CommitTransaction(); err != nil {
		dependencies.Logger.ErrorFunction(err, status)
		return fmt.Errorf("failed to commit transaction")
	}
	committed = true
	// Publish outside the transaction because Ably cannot participate in the database commit.
	channelName := helper.GetChatChannelName(phoneNumber.MetaPhoneNumberId, customer.Token)
	err = dependencies.Ably.Publish("status", channelName, dto_wa.NewMessageStatus(event))
	if err != nil {
		dependencies.Logger.ErrorFunction(err, channelName)
	}
	return nil
}

func insertIncomingMessage(ctx context.Context, dependencies *service.Dependencies, incomingMessage dto_wa.IncomingMessage, contacts []dto_wa.IncomingContact, metadata dto_wa.IncomingMetadata) error {
	timestamp, err := strconv.ParseInt(incomingMessage.Timestamp, 10, 64)
	if err != nil {
		dependencies.Logger.ErrorFunction(err, incomingMessage.Timestamp)
		return fmt.Errorf("incoming message timestamp invalid: %s", incomingMessage.Timestamp)
	}
	contact := helper.First(contacts, func(c dto_wa.IncomingContact) bool {
		return c.UserID == incomingMessage.FromUserID
	})
	if contact == nil {
		dependencies.Logger.ErrorFunction(fmt.Errorf("missing contact in WA incoming message: %s", incomingMessage.ID))
		return fmt.Errorf("missing contact in WA incoming message")
	}
	// skip whatsapp offical message
	if contact.Profile.Name == "WhatsApp Business" {
		return nil
	}
	transaction := dependencies.UnitOfWork.BeginTransaction()
	committed := false
	defer func() {
		if !committed {
			transaction.Rollback()
		}
	}()
	transactionDependencies := *dependencies
	transactionDependencies.UnitOfWork = transaction
	userPhoneNumber, err := transaction.WAPhoneNumberRepository().GetByMetaPhoneNumberId(ctx, metadata.PhoneNumberID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("user's phone number not found")
		}
		dependencies.Logger.ErrorFunction(err, metadata.PhoneNumberID)
		return fmt.Errorf("failed to get user's phone number")
	}
	// check customer exists based on waId or metaUserId
	existingCustomer, err := transaction.CustomerRepository().GetByWAIdOrMetaUserId(ctx, userPhoneNumber.UserId, contact.WaID, incomingMessage.FromUserID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("customer not found")
		}
		dependencies.Logger.ErrorFunction(err, userPhoneNumber.UserId, incomingMessage.FromUserID)
		return fmt.Errorf("failed to get customer")
	}
	// if no existing customer, create new
	var customerDTO dto_customer.Customer
	if existingCustomer == nil {
		var countryCode, phoneNumber string
		if incomingMessage.From != "" {
			cc, pn, err := helper.GetCountryCodeAndPhoneNumberFromWAId(incomingMessage.From)
			if err != nil {
				dependencies.Logger.ErrorFunction(err, metadata.DisplayPhoneNumber)
				cc = "."
				pn = incomingMessage.From
			}
			countryCode = cc
			phoneNumber = pn
		}
		createCustomer := feature_customer.Create{
			DisplayName:   contact.Profile.Name, // set same as wa display name
			WADisplayName: contact.Profile.Name,
			CountryCode:   countryCode,
			PhoneNumber:   phoneNumber,
			MetaUserId:    incomingMessage.FromUserID,
			WAId:          contact.WaID,
			Remarks:       "created from incoming WhatsApp message",
		}
		createCustomerResponse := createCustomer.Handle(ctx, &dto_account.User{DTOBase: dto.DTOBase{
			Id: userPhoneNumber.UserId,
		}}, &transactionDependencies)
		if !createCustomerResponse.Success {
			return errors.New(createCustomerResponse.Message)
		}
		customerDTO = *createCustomerResponse.Data
	} else { // update metaUserId if not exist
		var hasChange bool
		if incomingMessage.FromUserID != "" && existingCustomer.MetaUserId != incomingMessage.FromUserID {
			existingCustomer.MetaUserId = incomingMessage.FromUserID
			hasChange = true
		}
		if contact.WaID != "" && existingCustomer.WAId != contact.WaID {
			existingCustomer.WAId = contact.WaID
			hasChange = true
		}
		if contact.Profile.Name != "" && existingCustomer.WADisplayName != contact.Profile.Name {
			existingCustomer.WADisplayName = contact.Profile.Name
			hasChange = true
		}
		if hasChange {
			if err := transaction.CustomerRepository().Update(ctx, existingCustomer); err != nil {
				dependencies.Logger.ErrorFunction(err)
				return fmt.Errorf("failed update existing customer")
			}
		}
		customerDTO = dto_customer.NewCustomer(*existingCustomer)
	}
	message := dao_wa.Message{
		Sending:       false,
		CustomerId:    customerDTO.Id,
		PhoneNumberId: userPhoneNumber.Id,
		WAMessageId:   incomingMessage.ID,
		Timestamp:     timestamp,
		Type:          incomingMessage.Type,
		Payload:       incomingMessage.Payload,
		Token:         uuid.NewString(),
	}
	if err := transaction.WAMessageRepository().Insert(ctx, &message); err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return errors.New("duplicate message entry")
		}
		dependencies.Logger.ErrorFunction(err, message.CustomerId, message.PhoneNumberId, message.WAMessageId)
		return fmt.Errorf("failed to insert message")
	}
	if err := transaction.CommitTransaction(); err != nil {
		dependencies.Logger.ErrorFunction(err)
		return fmt.Errorf("failed to commit transaction")
	}
	committed = true
	messageDTO := dto_wa.NewMessage(message)
	channelName := helper.GetChatChannelName(metadata.PhoneNumberID, customerDTO.Token)
	err = dependencies.Ably.Publish("message", channelName, messageDTO)
	if err != nil {
		dependencies.Logger.ErrorFunction(err, channelName)
	}
	return nil
}

func RetrySendingMessage(ctx context.Context, messageId int32, dependencies *service.Dependencies) error {
	// if cannot get message, just return error
	var message *dao_wa.Message
	err := retry(ctx, 3, func() error {
		m, e := dependencies.UnitOfWork.WAMessageRepository().GetById(ctx, messageId)
		if e != nil {
			return e
		}
		message = m
		return nil
	})
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("message not found")
		}
		dependencies.Logger.ErrorFunction(err, messageId)
		return fmt.Errorf("failed to get message")
	}
	// if already exhausted retries, return
	if message.Attempts >= int32(cfg.Default().AliyunSMQ.MaxDequeueCount) {
		return nil
	}
	now := time.Now().UTC()
	// only process message if the status is rejected and has nextAttemptAt (http error or response no id) and nextAttemptAt has not reached yet
	if message.Status != types.WAMessageStatusRejected || message.NextAttemptAt == nil || message.NextAttemptAt.After(now) {
		return nil
	}
	// quickly update next_attempt_at to 10 min later so other worker won't pick it up
	// don't retry here, if failed, return
	updated, err := dependencies.UnitOfWork.WAMessageRepository().UpdateNextAttemptAt(ctx, message.Id, now.Add(10*time.Minute))
	if err != nil {
		dependencies.Logger.ErrorFunction(err, message.Id, now.Add(10*time.Minute))
		return fmt.Errorf("failed to update message next attempt at")
	}
	if !updated {
		return nil
	}
	// get phone number before everything else, becuase we need to send
	// notification to user using phone_number -> user_id
	var phoneNumber *dao_wa.PhoneNumber
	err = retry(ctx, 3, func() error {
		pn, e := dependencies.UnitOfWork.WAPhoneNumberRepository().GetById(ctx, message.PhoneNumberId)
		if e != nil {
			return e
		}
		phoneNumber = pn
		return nil
	})
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("phone number not found")
		}
		dependencies.Logger.ErrorFunction(err, message.PhoneNumberId)
		return fmt.Errorf("failed to get phone number")
	}
	// take a snapshot of errorMessages, so we know if there are new errors
	errorMessages := strings.Split(strings.TrimSpace(message.ErrorMessage), "\n")
	originalErrorMessages := strings.Split(strings.TrimSpace(message.ErrorMessage), "\n")
	defer func() {
		// no new error, send is success, return
		if len(errorMessages) == len(originalErrorMessages) {
			return
		}
		// send notification to user
		message.ErrorMessage = strings.Join(errorMessages, "\n")
		createNotification := feature_account_notification.Create{}
		if message.Attempts < int32(cfg.Default().AliyunSMQ.MaxDequeueCount) {
			message.Status = types.WAMessageStatusRejected
			minutes := 5 * message.Attempts
			message.NextAttemptAt = helper.ConvertToPointer(now.Add(time.Duration(minutes) * time.Minute))
			createNotification.Title = fmt.Sprintf("Error occurred while sending your message %d", message.Id)
			createNotification.Body = fmt.Sprintf("We have encountered an error while sending your message %d, will retry %d minutes later. Please check the errors below:\n\n%s", message.Id, minutes, strings.Join(errorMessages, "\n"))
			createNotification.Type = types.NotificationTypeWarning
		} else { // attemps exhausted
			message.Status = types.WAMessageStatusFailed
			message.NextAttemptAt = nil
			createNotification.Type = types.NotificationTypeError
			createNotification.Title = fmt.Sprintf("Failed to send your message %d", message.Id)
			createNotification.Body = fmt.Sprintf("We are sorry to inform you that we have failed to send your message %d despite %d attempts, please check the errors below:\n\n%s", message.Id, message.Attempts, strings.Join(errorMessages, "\n"))
		}
		err = retry(ctx, 3, func() error {
			e := dependencies.UnitOfWork.WAMessageRepository().Update(ctx, message)
			return e
		})
		// if no error, find out the user and send him/her notification
		if err == nil {
			var user *dao_account.User
			err = retry(ctx, 3, func() error {
				u, e := dependencies.UnitOfWork.AccountUserRepository().GetById(ctx, phoneNumber.UserId)
				if e != nil {
					dependencies.Logger.ErrorFunction(err, phoneNumber.UserId)
					return e
				}
				user = u
				return nil
			})
			if user != nil {
				// send user the notification, ignore any error
				_ = createNotification.Handle(ctx, helper.ConvertToPointer(dto_account.NewUser(*user)), dependencies)
			}
		} else {
			dependencies.Logger.ErrorFunction(err, message.Id)
		}
	}()
	// get business portfolio to get access token
	var businessPortfolio *dao_wa.BusinessPortfolio
	err = retry(ctx, 3, func() error {
		bp, _, e := dependencies.UnitOfWork.WAPhoneNumberRepository().GetBusinessPortfolioAndAccountByUserId(ctx, phoneNumber.UserId)
		if e != nil {
			return e
		}
		businessPortfolio = bp
		return nil
	})
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			errorMessages = append(errorMessages, "business portfolio not found")
			return fmt.Errorf("business portfolio not found")
		} else {
			errorMessages = append(errorMessages, "failed to get business portfolio")
			dependencies.Logger.ErrorFunction(err, phoneNumber.UserId)
			return fmt.Errorf("failed to get business portfolio")
		}
	}
	// send message using WhatsApp API
	response, err := dependencies.Whatsapp.SendMessage(ctx, phoneNumber.MetaPhoneNumberId, message.Payload, businessPortfolio.AccessToken)
	if err != nil {
		dependencies.Logger.ErrorFunction(err, phoneNumber.MetaPhoneNumberId)
		// WA errors are for user to see
		errorMessages = append(errorMessages, err.Error())
		return err
	} else if response == nil {
		err = fmt.Errorf("WhatsApp HTTP response is nil")
		dependencies.Logger.ErrorFunction(err, phoneNumber.MetaPhoneNumberId)
		errorMessages = append(errorMessages, err.Error())
		return err
	} else if len(response.Messages) == 0 {
		err = fmt.Errorf("no WhatsApp HTTP response messages")
		dependencies.Logger.ErrorFunction(err, phoneNumber.MetaPhoneNumberId)
		errorMessages = append(errorMessages, err.Error())
		return err
	} else if response.Messages[0].ID == "" {
		err = fmt.Errorf("message ID not returned by Meta")
		dependencies.Logger.ErrorFunction(err, phoneNumber.MetaPhoneNumberId)
		errorMessages = append(errorMessages, err.Error())
		return err
	}
	message.WAMessageId = strings.TrimSpace(response.Messages[0].ID)
	message.Attempts += 1
	message.Status = types.WAMessageStatusAccepted
	message.NextAttemptAt = nil
	err = retry(ctx, 3, func() error {
		e := dependencies.UnitOfWork.WAMessageRepository().Update(ctx, message)
		return e
	})
	if err != nil {
		dependencies.Logger.ErrorFunction(err, message.WAMessageId, message.Attempts, message.Status)
		return fmt.Errorf("failed to update message status")
	}
	return nil
}

func retry(ctx context.Context, attempts int, fn func() error) error {
	var err error
	for i := range attempts {
		if i > 0 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(time.Minute * time.Duration(i)):
			}
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if err = fn(); err == nil {
			return nil
		}
	}
	return err
}
