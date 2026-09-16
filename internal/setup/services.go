package setup

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/jjcheng/wawa-go/internal/cfg"
	dao_wa "github.com/jjcheng/wawa-go/internal/dao/wa"
	"github.com/jjcheng/wawa-go/internal/dto"
	dto_account "github.com/jjcheng/wawa-go/internal/dto/account"
	dto_customer "github.com/jjcheng/wawa-go/internal/dto/customer"
	dto_wa "github.com/jjcheng/wawa-go/internal/dto/wa"
	"github.com/jjcheng/wawa-go/internal/exception"
	feature_customer "github.com/jjcheng/wawa-go/internal/feature/customer"
	"github.com/jjcheng/wawa-go/internal/helper"
	"github.com/jjcheng/wawa-go/internal/repository"
	"github.com/jjcheng/wawa-go/internal/service"
	"github.com/jjcheng/wawa-go/internal/types"
	"gorm.io/gorm"
)

const webhookMessageVisibilityTimeoutSeconds = 180

// initializes and returns all application services
func SetupServices(unitOfWork repository.UnitOfWork, logger *service.Logger) *service.Dependencies {
	fileService := service.NewFileService(logger)
	messageQueueService := service.NewMessageQueue(logger)
	eventBridgeService := service.NewEventBridge(logger)
	ablyService := service.NewAbly(logger)
	whatsappService := service.NewWhatsapp(logger)
	dependencies := service.NewDependencies(unitOfWork, logger, fileService, messageQueueService, eventBridgeService, ablyService, whatsappService)
	return dependencies
}

func StartQueueListener(ctx context.Context, dependencies *service.Dependencies) {
	for {
		pollCtx, cancel := context.WithTimeout(ctx, time.Duration(cfg.Default().AliyunSMQ.PollingWaitSeconds+25)*time.Second)
		message, err := dependencies.MessageQueue.ReceiveMessage(pollCtx)
		cancel()
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			if errors.Is(err, context.DeadlineExceeded) {
				continue
			}
			dependencies.Logger.ErrorFunction(err)
			select {
			case <-ctx.Done():
				return
			case <-time.After(time.Second):
			}
			continue
		}
		if ctx.Err() != nil {
			return
		}
		if message == nil {
			continue
		}
		if err := dependencies.MessageQueue.ExtendMessageVisibility(message, webhookMessageVisibilityTimeoutSeconds); err != nil {
			dependencies.Logger.ErrorFunction(err, message.MessageID)
			continue
		}
		body := []byte(strings.TrimSpace(message.Body))
		if decodedBody, err := base64.StdEncoding.DecodeString(message.Body); err == nil && json.Valid(decodedBody) {
			body = decodedBody
		}
		queueJob, err := helper.DeserializeJSON[service.QueueJob](string(body))
		if err != nil {
			dependencies.Logger.ErrorFunction(err, message.MessageID)
			continue
		}
		if queueJob.Type == "wa_receive" {
			messageCtx, messageCancel := context.WithTimeout(context.Background(), 2*time.Minute)
			incoming, err := helper.DeserializeJSON[dto_wa.Incoming](string(queueJob.Data))
			if err != nil {
				messageCancel()
				dependencies.Logger.ErrorFunction(err, message.MessageID)
				continue
			}
			ex := processWAIncoming(messageCtx, dependencies, *incoming)
			messageCancel()
			if ex != nil {
				// delete the queued message if already stored
				if strings.Contains(ex.Message, "duplicate key value violates") {
					if deleteErr := dependencies.MessageQueue.DeleteMessage(message.ReceiptHandle); deleteErr != nil {
						dependencies.Logger.ErrorFunction(deleteErr, message.MessageID)
					}
				}
				continue
			}
			if err := dependencies.MessageQueue.DeleteMessage(message.ReceiptHandle); err != nil {
				dependencies.Logger.ErrorFunction(err, message.MessageID)
			}
		}

	}
}

func processWAIncoming(ctx context.Context, dependencies *service.Dependencies, incoming dto_wa.Incoming) *exception.Exception {
	for _, entry := range incoming.Entry {
		for _, change := range entry.Changes {
			for _, incomingMessage := range change.Value.Messages {
				if ex := storeWAIncomingMessage(ctx, dependencies, incomingMessage, change.Value.Contacts, change.Value.Metadata); ex != nil {
					if strings.Contains(ex.Message, "duplicate") {
						continue
					}
					return ex
				}
			}
			for _, status := range change.Value.Statuses {
				if ex := storeWAMessageStatus(ctx, dependencies, status); ex != nil {
					return ex
				}
			}
		}
	}
	return nil
}

func storeWAMessageStatus(ctx context.Context, dependencies *service.Dependencies, status dto_wa.Status) *exception.Exception {
	timestamp, err := strconv.ParseInt(status.Timestamp, 10, 64)
	if err != nil {
		return exception.NewCustomException(fmt.Sprintf("invalid WhatsApp status timestamp: %q", status.Timestamp), http.StatusBadGateway)
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
				return exception.NewCustomException(fmt.Sprintf("message not found with status ID: %s", status.ID), http.StatusNotFound)
			}
			return exception.NewCustomException(types.ExceptionMessageInternalServerError, http.StatusInternalServerError)
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
		return exception.NewCustomException(types.ExceptionMessageInternalServerError, http.StatusInternalServerError)
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
			return exception.NewCustomException(types.ExceptionMessageInternalServerError, http.StatusInternalServerError)
		}
	}
	customer, err := transaction.CustomerRepository().GetById(ctx, message.CustomerId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return exception.NewCustomException(fmt.Sprintf("customer not found with ID: %d", message.CustomerId), http.StatusNotFound)
		}
		return exception.NewCustomException(types.ExceptionMessageInternalServerError, http.StatusInternalServerError)
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
			return exception.NewCustomException(types.ExceptionMessageInternalServerError, http.StatusInternalServerError)
		}
	}
	phoneNumber, err := transaction.WAPhoneNumberRepository().GetById(ctx, message.PhoneNumberId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return exception.NewCustomException("phone number not found", http.StatusNotFound)
		}
		return exception.NewCustomException(types.ExceptionMessageInternalServerError, http.StatusInternalServerError)
	}
	channelName := helper.GetChatChannelName(phoneNumber.MetaPhoneNumberId, customer.Token)
	if err := transaction.CommitTransaction(); err != nil {
		dependencies.Logger.ErrorFunction(err, status)
		return exception.NewCustomException(types.ExceptionMessageInternalServerError, http.StatusInternalServerError)
	}
	committed = true
	// Publish outside the transaction because Ably cannot participate in the database commit.
	if err := dependencies.Ably.Publish("status", channelName, dto_wa.NewMessageStatusEvent(event)); err != nil {
		return exception.NewCustomException("error publishing Ably status event: "+channelName, http.StatusBadGateway)
	}
	return nil
}

func storeWAIncomingMessage(ctx context.Context, dependencies *service.Dependencies, incomingMessage dto_wa.IncomingMessage, contacts []dto_wa.IncomingContact, metadata dto_wa.IncomingMetadata) *exception.Exception {
	timestamp, err := strconv.ParseInt(incomingMessage.Timestamp, 10, 64)
	if err != nil {
		return exception.NewCustomException(fmt.Sprintf("invalid WhatsApp message timestamp: %q", incomingMessage.Timestamp), http.StatusBadGateway)
	}
	contact := helper.First(contacts, func(c dto_wa.IncomingContact) bool {
		return c.UserID == incomingMessage.FromUserID
	})
	if contact == nil {
		return exception.NewCustomException(fmt.Sprintf("missing contact in WA incoming message: %s", incomingMessage.ID), http.StatusBadGateway)
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
			return exception.NewCustomException(fmt.Sprintf("phone number ID not associated with a user: %s", metadata.PhoneNumberID), http.StatusNotFound)
		}
		return exception.NewCustomException(types.ExceptionMessageInternalServerError, http.StatusInternalServerError)
	}
	// check customer exists based on waId or metaUserId
	existingCustomer, err := transaction.CustomerRepository().GetByWAIdOrMetaUserId(ctx, userPhoneNumber.UserId, contact.WaID, incomingMessage.FromUserID)
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return exception.NewCustomException(types.ExceptionMessageInternalServerError, http.StatusInternalServerError)
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
			return exception.NewCustomException(createCustomerResponse.Message, createCustomerResponse.StatusCode)
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
				return exception.NewCustomException(types.ExceptionMessageInternalServerError, http.StatusInternalServerError)
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
			return exception.NewCustomException("duplicate message entry", http.StatusBadRequest)
		}
		return exception.NewCustomException(types.ExceptionMessageInternalServerError, http.StatusInternalServerError)
	}
	if err := transaction.CommitTransaction(); err != nil {
		return exception.NewCustomException(types.ExceptionMessageInternalServerError, http.StatusInternalServerError)
	}
	committed = true
	messageDTO := dto_wa.NewMessage(message)
	channelName := helper.GetChatChannelName(metadata.PhoneNumberID, customerDTO.Token)
	if err := dependencies.Ably.Publish("message", channelName, messageDTO); err != nil {
		return exception.NewCustomException("error publishing Ably message event: "+channelName, http.StatusBadGateway)
	}
	return nil
}
