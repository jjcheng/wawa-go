package feature_wa_message

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	dao_wa "github.com/jjcheng/wawa-go/internal/dao/wa"
	"github.com/jjcheng/wawa-go/internal/dto"
	dto_account "github.com/jjcheng/wawa-go/internal/dto/account"
	dto_customer "github.com/jjcheng/wawa-go/internal/dto/customer"
	dto_wa "github.com/jjcheng/wawa-go/internal/dto/wa"
	feature_customer "github.com/jjcheng/wawa-go/internal/feature/customer"
	"github.com/jjcheng/wawa-go/internal/helper"
	"github.com/jjcheng/wawa-go/internal/service"
	"github.com/jjcheng/wawa-go/internal/types"
	"gorm.io/gorm"
)

func ProcessIncoming(ctx context.Context, rawBody string, message *service.MessageQueueMessage, dependencies *service.Dependencies) error {
	messageCtx, messageCancel := context.WithTimeout(ctx, 10*time.Minute)
	defer messageCancel()
	incoming, err := helper.DeserializeJSON[dto_wa.Incoming](rawBody)
	if err != nil {
		dependencies.Logger.ErrorFunction(err, message.MessageID)
		return err
	}
	err = processWAIncoming(messageCtx, dependencies, *incoming)
	if err != nil {
		// delete the queued message if already stored
		if strings.Contains(err.Error(), "duplicate key value violates") {
			if deleteErr := dependencies.MessageQueue.DeleteMessage(message.ReceiptHandle); deleteErr != nil {
				dependencies.Logger.ErrorFunction(deleteErr, message.MessageID)
			}
		}
		return err
	}
	if message.ReceiptHandle != "" {
		if err := dependencies.MessageQueue.DeleteMessage(message.ReceiptHandle); err != nil {
			dependencies.Logger.ErrorFunction(err, message.MessageID)
		}
	}
	return nil
}

func processWAIncoming(ctx context.Context, dependencies *service.Dependencies, incoming dto_wa.Incoming) error {
	for _, entry := range incoming.Entry {
		for _, change := range entry.Changes {
			for _, incomingMessage := range change.Value.Messages {
				if err := storeWAIncomingMessage(ctx, dependencies, incomingMessage, change.Value.Contacts, change.Value.Metadata); err != nil {
					if strings.Contains(err.Error(), "duplicate") {
						continue
					}
					return err
				}
			}
			for _, status := range change.Value.Statuses {
				if err := storeWAMessageStatus(ctx, dependencies, status); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func storeWAMessageStatus(ctx context.Context, dependencies *service.Dependencies, status dto_wa.Status) error {
	timestamp, err := strconv.ParseInt(status.Timestamp, 10, 64)
	if err != nil {
		return err
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
			return err
		}
	}
	event := dao_wa.MessageStatusEvent{
		WAMessageId: status.ID,
		MessageId:   message.Id,
		Status:      messageStatus,
		Timestamp:   timestamp,
		Payload:     status.Payload,
	}
	if len(status.Errors) > 0 {
		var errorMessages []string
		for _, err := range status.Errors {
			if err.ErrorData != nil && err.ErrorData.Details != "" {
				errorMessages = append(errorMessages, err.ErrorData.Details)
			}
		}
		if len(errorMessages) > 0 {
			event.ErrorMessage = strings.Join(errorMessages, "\n")
		}
	}
	if err := transaction.WAMessageStatusEventRepository().Insert(ctx, &event); err != nil {
		return err
	}
	// Persist every callback for audit, but only advance the current status. Meta may
	// deliver callbacks out of order or omit delivered when a message is read directly.
	// This prevents stale sent, delivered, or failed callbacks from regressing read/played.
	if helper.CanTransitionWAMessageStatus(message.Status, messageStatus) {
		message.Status = messageStatus
		message.WAMessageId = status.ID // ensure waMessageId is there
		if status.Pricing != nil {
			message.Billable = status.Pricing.Billable
			message.BillingType = status.Pricing.Type
			message.Category = status.Pricing.Category
		}
		if err := transaction.WAMessageRepository().Update(ctx, message); err != nil {
			return err
		}
	}
	customer, err := transaction.CustomerRepository().GetById(ctx, message.CustomerId)
	if err != nil {
		return err
	}
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
			return err
		}
	}
	phoneNumber, err := transaction.WAPhoneNumberRepository().GetById(ctx, message.PhoneNumberId)
	if err != nil {
		return err
	}
	channelName := helper.GetChatChannelName(phoneNumber.MetaPhoneNumberId, customer.Token)
	if err := transaction.CommitTransaction(); err != nil {
		dependencies.Logger.ErrorFunction(err, status)
		return err
	}
	committed = true
	// Publish outside the transaction because Ably cannot participate in the database commit.
	_ = dependencies.Ably.Publish("status", channelName, dto_wa.NewMessageStatusEvent(event))
	return nil
}

func storeWAIncomingMessage(ctx context.Context, dependencies *service.Dependencies, incomingMessage dto_wa.IncomingMessage, contacts []dto_wa.IncomingContact, metadata dto_wa.IncomingMetadata) error {
	timestamp, err := strconv.ParseInt(incomingMessage.Timestamp, 10, 64)
	if err != nil {
		return err
	}
	contact := helper.First(contacts, func(c dto_wa.IncomingContact) bool {
		return c.UserID == incomingMessage.FromUserID
	})
	if contact == nil {
		return fmt.Errorf("missing contact in WA incoming message: %s", incomingMessage.ID)
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
			return fmt.Errorf("phone number ID not associated with a user: %s", metadata.PhoneNumberID)
		}
		return err
	}
	// check customer exists based on waId or metaUserId
	existingCustomer, err := transaction.CustomerRepository().GetByWAIdOrMetaUserId(ctx, userPhoneNumber.UserId, contact.WaID, incomingMessage.FromUserID)
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
	}
	// if no existing customer, create new
	var customerDTO dto_customer.Customer
	if existingCustomer == nil {
		countryCode, phoneNumber, err := helper.GetCountryCodeAndPhoneNumberFromWAId(incomingMessage.From)
		if err != nil {
			dependencies.Logger.ErrorFunction(err, metadata.DisplayPhoneNumber)
			countryCode = "."
			phoneNumber = incomingMessage.From
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
				return err
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
	}
	if err := transaction.WAMessageRepository().Insert(ctx, &message); err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return errors.New("duplicate message entry")
		}
		return err
	}
	if err := transaction.CommitTransaction(); err != nil {
		return err
	}
	committed = true
	messageDTO := dto_wa.NewMessage(message)
	channelName := helper.GetChatChannelName(metadata.PhoneNumberID, customerDTO.Token)
	_ = dependencies.Ably.Publish("message", channelName, messageDTO)
	return nil
}

func RetrySendingMessage(ctx context.Context, rawBody string, message *service.MessageQueueMessage, dependencies *service.Dependencies) error {
	if message == nil {
		return errors.New("message queue message is required")
	}
	job := dto_wa.RetrySendMessage{}
	if err := json.Unmarshal([]byte(rawBody), &job); err != nil {
		return err
	}
	if job.MessageId <= 0 {
		return fmt.Errorf("invalid message id in retry queue payload: %s", rawBody)
	}
	storedMessage, err := dependencies.UnitOfWork.WAMessageRepository().GetById(ctx, job.MessageId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			if message.ReceiptHandle != "" {
				if deleteErr := dependencies.MessageQueue.DeleteMessage(message.ReceiptHandle); deleteErr != nil {
					dependencies.Logger.ErrorFunction(deleteErr, message.MessageID)
				}
			}
			return nil
		}
		return err
	}
	if !shouldRetryWAMessageStatus(storedMessage.Status) {
		if message.ReceiptHandle != "" {
			if deleteErr := dependencies.MessageQueue.DeleteMessage(message.ReceiptHandle); deleteErr != nil {
				dependencies.Logger.ErrorFunction(deleteErr, message.MessageID)
			}
		}
		return nil
	}
	if storedMessage.PhoneNumberId == 0 {
		return fmt.Errorf("missing phone number id for retry message %d", storedMessage.Id)
	}
	phoneNumber, err := dependencies.UnitOfWork.WAPhoneNumberRepository().GetById(ctx, storedMessage.PhoneNumberId)
	if err != nil {
		return err
	}
	businessPortfolio, _, err := dependencies.UnitOfWork.WAPhoneNumberRepository().GetBusinessPortfolioAndAccountByUserId(ctx, phoneNumber.UserId)
	if err != nil {
		return err
	}
	if businessPortfolio == nil {
		return fmt.Errorf("missing business portfolio for retry message %d", storedMessage.Id)
	}
	if strings.TrimSpace(businessPortfolio.AccessToken) == "" {
		return fmt.Errorf("missing WhatsApp business portfolio access token for retry message %d", storedMessage.Id)
	}
	response, err := dependencies.Whatsapp.SendMessage(ctx, phoneNumber.MetaPhoneNumberId, storedMessage.Payload, businessPortfolio.AccessToken)
	if err != nil {
		storedMessage.Attempts += 1
		storedMessage.ErrorMessage = err.Error()
		storedMessage.Status = types.WAMessageStatusRejected
		storedMessage.NextAttemptAt = helper.ConvertToPointer(time.Now().UTC().Add(5 * time.Minute))
		if updateErr := dependencies.UnitOfWork.WAMessageRepository().Update(ctx, storedMessage); updateErr != nil {
			return updateErr
		}
		if message.ReceiptHandle != "" {
			if deleteErr := dependencies.MessageQueue.DeleteMessage(message.ReceiptHandle); deleteErr != nil {
				dependencies.Logger.ErrorFunction(deleteErr, message.MessageID)
			}
		}
		return err
	}
	if response != nil && len(response.Messages) > 0 && strings.TrimSpace(response.Messages[0].ID) != "" {
		storedMessage.WAMessageId = strings.TrimSpace(response.Messages[0].ID)
	}
	storedMessage.Attempts += 1
	storedMessage.ErrorMessage = ""
	storedMessage.Status = types.WAMessageStatusAccepted
	storedMessage.NextAttemptAt = nil
	if err := dependencies.UnitOfWork.WAMessageRepository().Update(ctx, storedMessage); err != nil {
		return err
	}
	if message.ReceiptHandle != "" {
		if deleteErr := dependencies.MessageQueue.DeleteMessage(message.ReceiptHandle); deleteErr != nil {
			dependencies.Logger.ErrorFunction(deleteErr, message.MessageID)
		}
	}
	return nil
}

func shouldRetryWAMessageStatus(status types.WAMessageStatus) bool {
	switch status {
	case types.WAMessageStatusRejected, types.WAMessageStatusFailed:
		return true
	default:
		return false
	}
}
