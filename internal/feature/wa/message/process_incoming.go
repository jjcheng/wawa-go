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
	dao_wa "github.com/jjcheng/wawa-go/internal/dao/wa"
	dto_customer "github.com/jjcheng/wawa-go/internal/dto/customer"
	dto_wa "github.com/jjcheng/wawa-go/internal/dto/wa"
	feature_account_notification "github.com/jjcheng/wawa-go/internal/feature/account/notification"
	feature_customer "github.com/jjcheng/wawa-go/internal/feature/customer"
	feature_wa_business_agent "github.com/jjcheng/wawa-go/internal/feature/wa/business_agent"
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
					return fmt.Errorf("invalid messages change value %v: %w", change.Value, err)
				}
				// process incoming messages
				for _, incomingMessage := range incomingValue.Messages {
					if err := insertIncomingMessage(ctx, dependencies, incomingMessage, incomingValue.Contacts, incomingValue.Metadata, false); err != nil {
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
			} else if change.Field == "standby" { // when business agent is active for a phone number
				var incomingValue dto_wa.IncomingValue
				if err := json.Unmarshal(change.Value, &incomingValue); err != nil {
					return fmt.Errorf("invalid messages change value %v: %w", change.Value, err)
				}
				// standby.messages and standby.contacts are only available if customer send in via business agent is active
				for _, message := range incomingValue.Standby.Messages {
					if err := insertIncomingMessage(ctx, dependencies, message, incomingValue.Standby.Contacts, incomingValue.Metadata, true); err != nil {
						// if already stored, it's not really an error
						if strings.Contains(err.Error(), "duplicate") {
							continue
						}
						return err
					}
				}
				// message echos is sent from the business agent
				for _, messageEcho := range incomingValue.Standby.MessageEchoes {
					if err := insertIncomingMessageEcho(ctx, dependencies, messageEcho, incomingValue.Metadata); err != nil {
						// if already stored, it's not really an error
						if strings.Contains(err.Error(), "duplicate") {
							continue
						}
						return err
					}
				}
				// check statuses
				for _, status := range incomingValue.Standby.Statuses {
					if err := insertIncomingStatus(ctx, dependencies, status); err != nil {
						return err
					}
				}
			} else if change.Field == "message_template_status_update" {
				var templateStatus dto_wa.IncomingTemplateStatusChange
				if err := json.Unmarshal(change.Value, &templateStatus); err != nil {
					return fmt.Errorf("invalid message template status change value value=%v error=%w", change.Value, err)
				}
				// if not approved but no reason, it maybe deleted
				if templateStatus.Event != "APPROVED" && templateStatus.Reason != "NONE" && templateStatus.Reason != "" {
					return nil
				}
				// do not store it into db, create a notification to inform user
				key := helper.GetTemplateStatusChangeCacheKey(fmt.Sprint(templateStatus.MessageTemplateId))
				cachedValue, err := dependencies.Cache.Get(ctx, key)
				if err != nil {
					// no need to return error, just skip
					dependencies.Logger.Error(err)
					return nil
				}
				userId, err := strconv.Atoi(cachedValue)
				if err != nil {
					// no need to return error, just skip
					dependencies.Logger.ErrorFunction(err, cachedValue)
					return nil
				}
				var title, body, url string
				var notificationType types.NotificationType
				var notificationIconType types.NotificationIconType
				var notificationCategory types.NotificationCategory
				if templateStatus.Event == "APPROVED" {
					title = fmt.Sprintf("Your template %s (%s) has been approved by Meta.", templateStatus.MessageTemplateName, templateStatus.MessageTemplateLanguage)
					body = "You can now go to Chats page, select at least 1 customer and start a broadcast with your new template."
					notificationType = types.NotificationTypeSuccess
					notificationIconType = types.NotificationIconTypeSuccess
					notificationCategory = types.NotificationCategoryHandsOff
					url = "/chats"
				} else {
					title = fmt.Sprintf("Your template %s (%s) status changed to %s by Meta.", templateStatus.Event, templateStatus.MessageTemplateName, templateStatus.MessageTemplateLanguage)
					body = fmt.Sprintf("Reason returned by Meta is: %s", templateStatus.Reason)
					notificationType = types.NotificationTypeWarning
					notificationIconType = types.NotificationIconTypeWarning
					notificationCategory = types.NotificationCategoryPending
					url = "/templates"
				}
				createNotification := feature_account_notification.Create{
					Category: notificationCategory,
					Type:     notificationType,
					IconType: notificationIconType,
					Title:    title,
					Body:     body,
					URL:      url,
					ToUserId: int32(userId),
				}
				_ = createNotification.Handle(ctx, dependencies)
				err = dependencies.Cache.Remove(ctx, key)
				if err != nil {
					dependencies.Logger.ErrorFunction(err, key)
				}
			} else if change.Field == "product_feed" {
				var productFeed dto_wa.IncomingProductFeed
				if err := json.Unmarshal(change.Value, &productFeed); err != nil {
					return fmt.Errorf("invalid message template status change.Value=%s: %w", change.Value, err)
				}
				if productFeed.Status == "finished" {
					// TODO: sync catalog db
				}
			} else if change.Field == "smb_message_echoes" { // messages sent by user from business app
				var incomingValue dto_wa.IncomingValue
				if err := json.Unmarshal(change.Value, &incomingValue); err != nil {
					return fmt.Errorf("invalid messages change value %v: %w", change.Value, err)
				}
				// process incoming messages echoes
				for _, incomingMessageEcho := range incomingValue.MessageEchoes {
					if err := insertIncomingMessageEcho(ctx, dependencies, incomingMessageEcho, incomingValue.Metadata); err != nil {
						// if already stored, it's not really an error
						if strings.Contains(err.Error(), "duplicate") {
							continue
						}
						return err
					}
				}
				// proces statuses
				for _, status := range incomingValue.Statuses {
					if err := insertIncomingStatus(ctx, dependencies, status); err != nil {
						return err
					}
				}
			} else if change.Field == "smb_app_state_sync" {

			} else if change.Field == "messaging_handovers" {
				var incomingValue dto_wa.IncomingValue
				if err := json.Unmarshal(change.Value, &incomingValue); err != nil {
					return fmt.Errorf("invalid messages change value %v: %w", change.Value, err)
				}
				if incomingValue.IncomingHandovers != nil {
					return handoverMessaging(ctx, dependencies, incomingValue)
				}
			}
		}
	}
	return nil
}

func insertIncomingStatus(ctx context.Context, dependencies *service.Dependencies, status dto_wa.Status) error {
	timestamp, err := strconv.ParseInt(status.Timestamp, 10, 64)
	if err != nil {
		return fmt.Errorf("insert message status timestamp invalid timestamp=%s error=%w", status.Timestamp, err)
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
			return fmt.Errorf("failed to get message from status messageId=%s error=%w", status.ID, err)
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
		return fmt.Errorf("failed to insert message status waMessageId=%s messageId=%d status=%s timestamp=%v error=%w", event.WAMessageId, event.MessageId, event.Status, event.Timestamp, err)
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
			return fmt.Errorf("failed to update message status=%s waMessageId=%s errorMessage=%s billable=%v billingType=%s category=%s error=%w", message.Status, message.WAMessageId, message.ErrorMessage, message.Billable, message.BillingType, message.Category, err)
		}
	}
	// get customer
	customer, err := transaction.CustomerRepository().GetById(ctx, message.CustomerId)
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("failed to get customer customerId=%d error=%w", message.CustomerId, err)
		}
		return fmt.Errorf("customer not found %d", message.CustomerId)
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
			return fmt.Errorf("failed update customer Meta user id or WA id customerId=%d error=%w", customer.Id, err)
		}
	}
	// get phone number
	phoneNumber, err := transaction.WAPhoneNumberRepository().GetById(ctx, message.PhoneNumberId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("phone number not found phoneNumberId=%d", message.PhoneNumberId)
		}
		return fmt.Errorf("failed to get phone number phoneNumberId=%d error=%w", message.PhoneNumberId, err)
	}
	// now commit the transaction
	if err := transaction.CommitTransaction(); err != nil {
		return fmt.Errorf("failed to commit transaction error=%w", err)
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

func insertIncomingMessage(ctx context.Context, dependencies *service.Dependencies, incomingMessage dto_wa.IncomingMessage, contacts []dto_wa.IncomingContact, metadata dto_wa.IncomingMetadata, isStandby bool) error {
	timestamp, err := strconv.ParseInt(incomingMessage.Timestamp, 10, 64)
	if err != nil {
		return fmt.Errorf("incoming message timestamp invalid: %s", incomingMessage.Timestamp)
	}
	contact := helper.First(contacts, func(c dto_wa.IncomingContact) bool {
		return c.UserID == incomingMessage.FromUserID
	})
	if contact == nil {
		return fmt.Errorf("missing contact in WA incoming message: %s", incomingMessage.FromUserID)
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
			return fmt.Errorf("user's phone number not found %s", metadata.PhoneNumberID)
		}
		return fmt.Errorf("failed to get user's phone number %s: %w", metadata.PhoneNumberID, err)
	}
	// check customer exists based on waId or metaUserId
	existingCustomer, err := transaction.CustomerRepository().GetByWAIdOrMetaUserId(ctx, userPhoneNumber.Id, contact.WaID, incomingMessage.FromUserID)
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("failed to get customer fromUserId=%s: %w", incomingMessage.FromUserID, err)
		}
	}
	// if no existing customer, create new
	var customerDTO dto_customer.Customer
	if existingCustomer == nil {
		var countryCode, phoneNumber string
		if incomingMessage.From != "" {
			cc, pn, err := helper.GetCountryCodeAndPhoneNumberFromWAId(incomingMessage.From)
			if err != nil {
				dependencies.Logger.ErrorFunction(err)
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
			PhoneNumberId: userPhoneNumber.Id,
			FromIncoming:  true,
			AgentEnabled:  true,
		}
		createCustomerResponse := createCustomer.Handle(ctx, nil, &transactionDependencies)
		if !createCustomerResponse.Success {
			return createCustomerResponse.Error
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
				return fmt.Errorf("failed update existing customer %d: %w", existingCustomer.Id, err)
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
		if errors.Is(err, gorm.ErrDuplicatedKey) || strings.Contains(err.Error(), "duplicate key value violates unique constraint") {
			return errors.New("duplicate message entry")
		}
		return fmt.Errorf("failed to insert message customerId=%d phoneNumberId=%d waMessageId=%s timestamp=%v error=%w", message.CustomerId, userPhoneNumber.Id, message.WAMessageId, message.Timestamp, err)
	}
	if err := transaction.CommitTransaction(); err != nil {
		return fmt.Errorf("failed to commit transaction error=%w", err)
	}
	committed = true
	messageDTO := dto_wa.NewMessage(message)
	messageDTO.CustomerName = customerDTO.DisplayName
	// publish to the chat room between phone number and customer
	chatChannelName := helper.GetChatChannelName(metadata.PhoneNumberID, customerDTO.Token)
	err = dependencies.Ably.Publish("message", chatChannelName, messageDTO)
	if err != nil {
		dependencies.Logger.ErrorFunction(err, chatChannelName)
	}
	// publish to phone number channel
	phoneNumberChannelName := helper.GetPhoneNumberChannelName(customerDTO.PhoneNumberId)
	err = dependencies.Ably.Publish("phone_number_message", phoneNumberChannelName, messageDTO)
	if err != nil {
		dependencies.Logger.ErrorFunction(err, phoneNumberChannelName)
	}
	// if customer can be handled by business agent and this phone number is currently running agent, pass control to business agent
	// if is from standby, it's already handled by business agent
	if !isStandby && customerDTO.AgentEnabled && userPhoneNumber.AgentEnabled {
		passControl := feature_wa_business_agent.PassControl{
			CustomerId:  message.CustomerId,
			ToAgent:     true,
			FromWebhook: true,
		}
		response := passControl.Handle(ctx, nil, dependencies)
		if !response.Success {
			dependencies.Logger.ErrorFunction(response.Error, message.CustomerId)
		}
	}
	// send notifications to user if message contains keywords
	if messageText := messageDTO.Text(); messageText != "" {
		if err := notifyMatchingBusinessAgentKeywords(ctx, dependencies, userPhoneNumber.Id, customerDTO, messageText); err != nil {
			dependencies.Logger.ErrorFunction(err, userPhoneNumber.Id, customerDTO.Id, message.Id)
		}
	}
	return nil
}

func notifyMatchingBusinessAgentKeywords(ctx context.Context, dependencies *service.Dependencies, phoneNumberId int32, customer dto_customer.Customer, messageText string) error {
	matchedKeywords, err := dependencies.UnitOfWork.WABusinessAgentKeywordRepository().ListMatchingByPhoneNumberId(ctx, phoneNumberId, messageText)
	if err != nil {
		return fmt.Errorf("failed to list matching business agent keywords phoneNumberId=%d: %w", phoneNumberId, err)
	}
	if len(matchedKeywords) == 0 {
		return nil
	}
	// send notification to all users managing this phone number
	userPhoneNumbers, err := dependencies.UnitOfWork.AccountUserPhoneNumberRepository().ListByPhoneNumberId(ctx, phoneNumberId)
	if err != nil {
		return fmt.Errorf("failed to list phone number users phoneNumberId=%d: %w", phoneNumberId, err)
	}
	var notificationErrors []error
	for _, keyword := range matchedKeywords {
		for _, userPhoneNumber := range userPhoneNumbers {
			createNotification := feature_account_notification.Create{
				ToUserId: userPhoneNumber.UserId,
				Category: types.NotificationCategoryPending,
				IconType: types.NotificationIconTypeChat,
				Type:     types.NotificationTypeInfo,
				Title:    keyword.NotificationTitle,
				Body:     fmt.Sprintf("Keyword %q detected in a message with customer %s.", keyword.Keyword, customer.DisplayName),
				URL:      fmt.Sprintf("/chats/%d/chat", customer.Id),
			}
			if response := createNotification.Handle(ctx, dependencies); !response.Success {
				notificationErrors = append(notificationErrors, fmt.Errorf("failed to notify userId=%d keywordId=%d: %w", userPhoneNumber.UserId, keyword.Id, response.Error))
			}
		}
	}
	return errors.Join(notificationErrors...)
}

func insertIncomingMessageEcho(ctx context.Context, dependencies *service.Dependencies, incomingMessageEcho dto_wa.IncomingMessageStandbyEcho, metadata dto_wa.IncomingMetadata) error {
	timestamp, err := strconv.ParseInt(incomingMessageEcho.Timestamp, 10, 64)
	if err != nil {
		return fmt.Errorf("incoming message echo timestamp invalid: %s", incomingMessageEcho.Timestamp)
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
			return fmt.Errorf("user's phone number not found %s", metadata.PhoneNumberID)
		}
		return fmt.Errorf("failed to get user's phone number %s: %w", metadata.PhoneNumberID, err)
	}
	// check customer exists based on waId or metaUserId
	existingCustomer, err := transaction.CustomerRepository().GetByWAIdOrMetaUserId(ctx, userPhoneNumber.Id, incomingMessageEcho.Message.To, incomingMessageEcho.Message.Recipient)
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("failed to get customer error=%w", err)
		}
	}
	// if no existing customer, create new
	var customerDTO dto_customer.Customer
	if existingCustomer == nil {
		var countryCode, phoneNumber string
		if incomingMessageEcho.Message.To != "" {
			cc, pn, err := helper.GetCountryCodeAndPhoneNumberFromWAId(incomingMessageEcho.Message.To)
			if err != nil {
				dependencies.Logger.ErrorFunction(err)
				cc = "."
				pn = incomingMessageEcho.Message.To
			}
			countryCode = cc
			phoneNumber = pn
		}
		createCustomer := feature_customer.Create{
			DisplayName:   "Unknown name",
			WADisplayName: "Unknown name",
			CountryCode:   countryCode,
			PhoneNumber:   phoneNumber,
			MetaUserId:    incomingMessageEcho.Message.Recipient,
			WAId:          incomingMessageEcho.Message.To,
			Remarks:       "created from incoming WhatsApp message echo",
			PhoneNumberId: userPhoneNumber.Id,
			FromIncoming:  true,
		}
		createCustomerResponse := createCustomer.Handle(ctx, nil, &transactionDependencies)
		if !createCustomerResponse.Success {
			return createCustomerResponse.Error
		}
		customerDTO = *createCustomerResponse.Data
	} else {
		var hasChange bool
		if incomingMessageEcho.Message.Recipient != "" && existingCustomer.MetaUserId != incomingMessageEcho.Message.Recipient {
			existingCustomer.MetaUserId = incomingMessageEcho.Message.Recipient
			hasChange = true
		}
		if incomingMessageEcho.Message.To != "" && existingCustomer.WAId != incomingMessageEcho.Message.To {
			existingCustomer.WAId = incomingMessageEcho.Message.To
			hasChange = true
		}
		if hasChange {
			if err := transaction.CustomerRepository().Update(ctx, existingCustomer); err != nil {
				return fmt.Errorf("failed update existing customer %d: %w", existingCustomer.Id, err)
			}
		}
		customerDTO = dto_customer.NewCustomer(*existingCustomer)
	}
	message := dao_wa.Message{
		Sending:       true, // message echos are messages sent by business agent
		CustomerId:    customerDTO.Id,
		PhoneNumberId: userPhoneNumber.Id,
		WAMessageId:   incomingMessageEcho.ID,
		Timestamp:     timestamp,
		Type:          incomingMessageEcho.Message.Type,
		Payload:       incomingMessageEcho.Payload["message"].(map[string]any),
		Token:         uuid.NewString(),
		ByAgent:       true,
	}
	if err := transaction.WAMessageRepository().Insert(ctx, &message); err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) || strings.Contains(err.Error(), "duplicate key value violates unique constraint") {
			return errors.New("duplicate message entry")
		}
		return fmt.Errorf("failed to insert message customerId=%d phoneNumberId=%d waMessageId=%s timestamp=%v error=%w", message.CustomerId, userPhoneNumber.Id, message.WAMessageId, message.Timestamp, err)
	}
	if err := transaction.CommitTransaction(); err != nil {
		return fmt.Errorf("failed to commit transaction error=%w", err)
	}
	committed = true
	// send notifications to user if message contains keywords
	messageDTO := dto_wa.NewMessage(message)
	if messageText := messageDTO.Text(); messageText != "" {
		if err := notifyMatchingBusinessAgentKeywords(ctx, dependencies, userPhoneNumber.Id, customerDTO, messageText); err != nil {
			dependencies.Logger.ErrorFunction(err, userPhoneNumber.Id, customerDTO.Id, message.Id)
		}
	}
	return nil
}

func handoverMessaging(ctx context.Context, dependencies *service.Dependencies, incomingValue dto_wa.IncomingValue) error {
	// get phone number
	phoneNumber, err := dependencies.UnitOfWork.WAPhoneNumberRepository().GetByMetaPhoneNumberId(ctx, incomingValue.IncomingHandovers.Recipient.PhoneNumberID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return fmt.Errorf("failed to get phone number by meta phone number id, phoneNumberId=%s error=%w", incomingValue.IncomingHandovers.Recipient.PhoneNumberID, err)
	}
	// get customer
	customer, err := dependencies.UnitOfWork.CustomerRepository().GetByWAIdOrMetaUserId(ctx, phoneNumber.Id, incomingValue.Sender.PhoneNumber, incomingValue.Sender.PhoneNumber)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return fmt.Errorf("failed by get customer by waId of metaUserId, phoneNumberId=%s error=%w", incomingValue.IncomingHandovers.Recipient.PhoneNumberID, err)
	}
	customer.AgentEnabled = false // disable the agent, must be activated again manually
	if err := dependencies.UnitOfWork.CustomerRepository().Update(ctx, customer); err != nil {
		return fmt.Errorf("failed update customer agent_enabled, customerId=%d error=%w", customer.Id, err)
	}
	// get reason
	var metadata string
	var summary string
	if incomingValue.IncomingHandovers.Type == "control_passed" {
		metadata = incomingValue.IncomingHandovers.ControlPassed.Metadata
		if incomingValue.IncomingHandovers.ControlPassed.ConversationContext != nil && incomingValue.IncomingHandovers.ControlPassed.ConversationContext.Type == "summary" && incomingValue.IncomingHandovers.ControlPassed.ConversationContext.Summary != nil {
			summary = incomingValue.IncomingHandovers.ControlPassed.ConversationContext.Summary.Text
		}
	} else {
		metadata = incomingValue.IncomingHandovers.ControlTaken.Metadata
		if incomingValue.IncomingHandovers.ControlTaken.ConversationContext != nil && incomingValue.IncomingHandovers.ControlTaken.ConversationContext.Type == "summary" && incomingValue.IncomingHandovers.ControlTaken.ConversationContext.Summary != nil {
			summary = incomingValue.IncomingHandovers.ControlTaken.ConversationContext.Summary.Text
		}
	}
	// send notifications to all users managing this phone number
	userPhoneNumbers, err := dependencies.UnitOfWork.AccountUserPhoneNumberRepository().ListByPhoneNumberId(ctx, phoneNumber.Id)
	if err != nil {
		return fmt.Errorf("failed to list userPhoneNumbers by phoneNumberId, phoneNumberId=%d error=%w", phoneNumber.Id, err)
	}
	if len(userPhoneNumbers) == 0 {
		return nil
	}
	createNotification := feature_account_notification.Create{
		Category: types.NotificationCategoryPending,
		IconType: types.NotificationIconTypeChat,
		Type:     types.NotificationTypeWarning,
		Title:    fmt.Sprintf("Business Agent has handed over a conversation to you: %s", metadata),
		Body:     fmt.Sprintf("Conversation with %s", customer.DisplayName),
		URL:      fmt.Sprintf("/chats/%d/chat", customer.Id),
	}
	if summary != "" {
		createNotification.Body += fmt.Sprintf("\nSummary: %s", summary)
	}
	createNotification.Body += "\n\nYou have to manually enable business agent for this customer again."
	for _, userPhoneNumber := range userPhoneNumbers {
		createNotification.ToUserId = userPhoneNumber.UserId
		_ = createNotification.Handle(ctx, dependencies)
		if summary != "" {
			create := Create{
				Type: MessageTypeText,
				Text: &TextObject{
					Body: fmt.Sprintf("ℹ️ *%s*\n\n%s\n\nYou have to manually enable business agent for this customer again.", strings.ReplaceAll(createNotification.Title, "_", " "), summary),
				},
			}
			payload, err := messagePayload(create)
			if err != nil {
				return fmt.Errorf("failed to create message payload: %w", err)
			}
			payload["messaging_product"] = "whatsapp"
			payload["recipient_type"] = "individual"
			// use meta_user_id to send if not empty
			if customer.WAId != "" {
				payload["to"] = customer.WAId
			} else {
				payload["to"] = customer.MetaUserId
			}
			delete(payload, "customer_id")
			token := uuid.NewString()
			payload["biz_opaque_callback_data"] = token
			newMessage := dao_wa.Message{
				Sending:       true,
				SenderUserId:  &userPhoneNumber.UserId,
				PhoneNumberId: customer.PhoneNumberId,
				CustomerId:    customer.Id,
				WAMessageId:   uuid.NewString(),
				Timestamp:     time.Now().Unix(),
				Type:          "text",
				Status:        types.WAMessageStatusDelivered,
				Token:         uuid.NewString(),
				ByAgent:       true,
				Payload:       payload,
			}
			err = dependencies.UnitOfWork.WAMessageRepository().Insert(ctx, &newMessage)
			if err != nil {
				dependencies.Logger.ErrorFunction(err)
			}
		}
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
		return fmt.Errorf("failed to get message messageId=%d error=%w", messageId, err)
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
	nextAttemptAt := now.Add(10 * time.Minute)
	updated, err := dependencies.UnitOfWork.WAMessageRepository().UpdateNextAttemptAt(ctx, message.Id, nextAttemptAt)
	if err != nil {
		return fmt.Errorf("failed to update message next attempt at nextAttemptAt=%v messageId=%d error=%w", nextAttemptAt, message.Id, err)
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
			return fmt.Errorf("phone number not found phoneNumberId=%d error=%w", message.PhoneNumberId, err)
		}
		return fmt.Errorf("failed to get phone number phoneNumberId=%d error=%w", message.PhoneNumberId, err)
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
		createNotification := feature_account_notification.Create{
			ToUserId: *message.SenderUserId,
		}
		if message.Attempts < int32(cfg.Default().AliyunSMQ.MaxDequeueCount) {
			message.Status = types.WAMessageStatusRejected
			minutes := 5 * message.Attempts
			message.NextAttemptAt = helper.ConvertToPointer(now.Add(time.Duration(minutes) * time.Minute))
			createNotification.Title = fmt.Sprintf("Warning: Error occurred while sending your message %d.", message.Id)
			createNotification.Body = fmt.Sprintf("We have encountered an error while sending your message %d, we will retry in %d minutes later. Please check the errors:\n\n%s", message.Id, minutes, strings.Join(errorMessages, "\n"))
			createNotification.Type = types.NotificationTypeWarning
			createNotification.IconType = types.NotificationIconTypeWarning
			createNotification.Category = types.NotificationCategoryHandsOff
		} else { // attemps exhausted
			message.Status = types.WAMessageStatusFailed
			message.NextAttemptAt = nil
			createNotification.Type = types.NotificationTypeError
			createNotification.IconType = types.NotificationIconTypeError
			createNotification.Category = types.NotificationCategoryPending
			createNotification.Title = fmt.Sprintf("Error: Failed to send your message %d.", message.Id)
			createNotification.Body = fmt.Sprintf("We are sorry to inform you that we have failed to send your message %d despite %d attempts, please check the errors:\n\n%s", message.Id, message.Attempts, strings.Join(errorMessages, "\n"))
		}
		err = retry(ctx, 3, func() error {
			e := dependencies.UnitOfWork.WAMessageRepository().Update(ctx, message)
			return e
		})
		// if no error, find out the user and send him/her notification
		if err == nil {
			// send user the notification, ignore any error
			_ = createNotification.Handle(ctx, dependencies)
		} else {
			dependencies.Logger.ErrorFunction(err, message.Id)
		}
	}()
	// get business portfolio to get access token
	if message.SenderUserId == nil {
		return fmt.Errorf("missing sender user id in message")
	}
	var businessPortfolio *dao_wa.BusinessPortfolio
	err = retry(ctx, 3, func() error {
		bp, _, e := dependencies.UnitOfWork.WAPhoneNumberRepository().GetBusinessPortfolioAndAccountByUserId(ctx, *message.SenderUserId)
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
			return fmt.Errorf("failed to get business portfolio userId=%d error=%w", *message.SenderUserId, err)
		}
	}
	// send message using WhatsApp API
	response, err := dependencies.Whatsapp.SendMessage(ctx, phoneNumber.MetaPhoneNumberId, message.Payload, businessPortfolio.AccessToken)
	if err != nil {
		// WA errors are for user to see
		errorMessages = append(errorMessages, err.Error())
		return err
	} else if response == nil {
		err = fmt.Errorf("WhatsApp HTTP response is nil")
		errorMessages = append(errorMessages, err.Error())
		return err
	} else if len(response.Messages) == 0 {
		err = fmt.Errorf("no WhatsApp HTTP response messages")
		errorMessages = append(errorMessages, err.Error())
		return err
	} else if response.Messages[0].ID == "" {
		err = fmt.Errorf("message ID not returned by Meta")
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
		return fmt.Errorf("failed to update message status waMessageId=%s attempts=%d status=%s error=%w", message.WAMessageId, message.Attempts, message.Status, err)
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
