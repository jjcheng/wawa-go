package setup

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jjcheng/wawa-go/internal/cfg"
	dao_wa "github.com/jjcheng/wawa-go/internal/dao/wa"
	"github.com/jjcheng/wawa-go/internal/dto"
	dto_account "github.com/jjcheng/wawa-go/internal/dto/account"
	dto_wa "github.com/jjcheng/wawa-go/internal/dto/wa"
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
	ablyService := service.NewAbly(logger)
	whatsappService := service.NewWhatsapp(logger)
	dependencies := service.NewDependencies(unitOfWork, logger, fileService, messageQueueService, ablyService, whatsappService)
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
		incoming, err := helper.DeserializeJSON[dto_wa.Incoming](string(body))
		if err != nil {
			dependencies.Logger.ErrorFunction(err, message.MessageID)
			continue
		}
		messageCtx, messageCancel := context.WithTimeout(context.Background(), 2*time.Minute)
		err = processWAIncoming(messageCtx, dependencies, *incoming)
		messageCancel()
		if err != nil {
			// delete the queued message if already stored
			if strings.Contains(err.Error(), "duplicate key value violates") {
				if err := dependencies.MessageQueue.DeleteMessage(message.ReceiptHandle); err != nil {
					dependencies.Logger.ErrorFunction(err, message.MessageID)
				}
			}
			continue
		}
		if err := dependencies.MessageQueue.DeleteMessage(message.ReceiptHandle); err != nil {
			dependencies.Logger.ErrorFunction(err, message.MessageID)
		}
	}
}

func processWAIncoming(ctx context.Context, dependencies *service.Dependencies, incoming dto_wa.Incoming) error {
	for _, entry := range incoming.Entry {
		for _, change := range entry.Changes {
			for _, incomingMessage := range change.Value.Messages {
				if err := storeWAIncomingMessage(ctx, dependencies, incomingMessage, change.Value.Contacts, change.Value.Metadata); err != nil {
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
		return fmt.Errorf("invalid WhatsApp status timestamp %q: %w", status.Timestamp, err)
	}
	transaction := dependencies.UnitOfWork.BeginTransaction()
	committed := false
	defer func() {
		if !committed {
			transaction.Rollback()
		}
	}()
	messageStatus := types.WAMessageStatus(status.Status)
	message, err := transaction.WAMessageRepository().GetByWAMessageId(ctx, status.ID)
	if err != nil {
		return err
	}
	event := dao_wa.MessageStatusEvent{
		WAMessageId: status.ID,
		Status:      messageStatus,
		Timestamp:   timestamp,
		Payload:     status.Payload,
	}
	if err := transaction.WAMessageStatusEventRepository().Insert(ctx, &event); err != nil {
		return err
	}
	// Persist every callback for audit, but only advance the current status. Meta may
	// deliver callbacks out of order or omit delivered when a message is read directly.
	// This prevents stale sent, delivered, or failed callbacks from regressing read/played.
	if helper.CanTransitionWAMessageStatus(message.Status, messageStatus) {
		message.Status = messageStatus
		message.CustomerMetaUserId = status.RecipientUserID
		message.CustomerWAId = status.RecipientID
		if status.Pricing != nil {
			message.Billable = status.Pricing.Billable
			message.BillingType = status.Pricing.Type
			message.Category = status.Pricing.Category
		}
		transaction.WAMessageRepository().Update(ctx, message)
	}
	if err := transaction.CommitTransaction(); err != nil {
		return err
	}
	committed = true
	err = dependencies.Ably.Publish("status", helper.GetChatChannelName(message.PhoneNumberId, message.CustomerWAId, message.CustomerMetaUserId), dto_wa.NewMessageStatusEvent(event))
	return err
}

func storeWAIncomingMessage(ctx context.Context, dependencies *service.Dependencies, incomingMessage dto_wa.IncomingMessage, contacts []dto_wa.IncomingContact, metadata dto_wa.IncomingMetadata) error {
	timestamp, err := strconv.ParseInt(incomingMessage.Timestamp, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid WhatsApp message timestamp %q: %w", incomingMessage.Timestamp, err)
	}
	contact := helper.First(contacts, func(c dto_wa.IncomingContact) bool {
		return c.UserID == incomingMessage.FromUserID
	})
	if contact == nil {
		return fmt.Errorf("missing contact in WA incoming message: %s", incomingMessage.ID)
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
	message := dao_wa.Message{
		Sending:            false,
		PhoneNumber:        metadata.DisplayPhoneNumber,
		PhoneNumberId:      metadata.PhoneNumberID,
		CustomerName:       contact.Profile.Name,
		CustomerWAId:       contact.WaID,
		CustomerMetaUserId: incomingMessage.FromUserID,
		WAMessageId:        incomingMessage.ID,
		Timestamp:          timestamp,
		Type:               incomingMessage.Type,
		Payload:            incomingMessage.Payload,
	}
	if err := transaction.WAMessageRepository().Insert(ctx, &message); err != nil {
		return err
	}
	userPhoneNumber, err := transaction.WAPhoneNumberRepository().GetByPhoneNumberId(ctx, metadata.PhoneNumberID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("phone number id is not associated with a user")
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
	if existingCustomer == nil {
		countryCode, phoneNumber, err := helper.GetCountryCodeAndPhoneNumberFromWAId(incomingMessage.From)
		if err != nil {
			dependencies.Logger.ErrorFunction(err, metadata.DisplayPhoneNumber)
			// use a fake country code + phone number so it does not violate unique rule
			countryCode = "."
			phoneNumber = strings.ReplaceAll(uuid.NewString(), "-", "")
		}
		createCustomer := feature_customer.Create{
			DisplayName: contact.Profile.Name,
			CountryCode: countryCode,
			PhoneNumber: phoneNumber,
			MetaUserId:  incomingMessage.FromUserID,
			WAId:        contact.WaID,
			Remarks:     "automatically created by incoming WhatsApp message",
		}
		createCustomerResponse := createCustomer.Handle(ctx, &dto_account.User{DTOBase: dto.DTOBase{
			Id: userPhoneNumber.UserId,
		}}, &transactionDependencies)
		if !createCustomerResponse.Success {
			return errors.New(createCustomerResponse.Message)
		}
	} else { // update metaUserId if not exist
		var hasChange bool
		if existingCustomer.MetaUserId != incomingMessage.FromUserID {
			existingCustomer.MetaUserId = incomingMessage.FromUserID
			hasChange = true
		}
		if existingCustomer.WAId != contact.WaID {
			existingCustomer.WAId = contact.WaID
			hasChange = true
		}
		if hasChange {
			if err := transaction.CustomerRepository().Update(ctx, existingCustomer); err != nil {
				return err
			}
		}
	}
	if err := transaction.CommitTransaction(); err != nil {
		return err
	}
	committed = true
	messageDTO := dto_wa.NewMessage(message)
	return dependencies.Ably.Publish("message", helper.GetChatChannelName(message.PhoneNumberId, message.CustomerWAId, message.CustomerMetaUserId), messageDTO)
}
