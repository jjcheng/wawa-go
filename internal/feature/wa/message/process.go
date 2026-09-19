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

// do not parse rawBody to dto_wa.Incoming before here as we are receiving raw data from Meta
func ProcessIncoming(ctx context.Context, incoming dto_wa.Incoming, message *service.MessageQueueMessage, dependencies *service.Dependencies) error {
	dependencies.Logger.Infof("processing WA incoming: %s", message.MessageID)
	acknowledge := func() {
		if message.ReceiptHandle == "" {
			return
		}
		if deleteErr := dependencies.MessageQueue.DeleteMessage(message.ReceiptHandle); deleteErr != nil {
			dependencies.Logger.ErrorFunction(deleteErr, message.MessageID)
		}
	}
	err := processIncoming(ctx, dependencies, incoming)
	if err != nil {
		// delete the queued message if already stored
		if strings.Contains(err.Error(), "duplicate key value violates") {
			acknowledge()
		}
		return err
	}
	acknowledge()
	return nil
}

func processIncoming(ctx context.Context, dependencies *service.Dependencies, incoming dto_wa.Incoming) error {
	for _, entry := range incoming.Entry {
		for _, change := range entry.Changes {
			if change.Field == "messages" {
				var incomingValue dto_wa.IncomingValue
				if err := json.Unmarshal(change.Value, &incomingValue); err != nil {
					return fmt.Errorf("invalid messages change value: %w", err)
				}
				for _, incomingMessage := range incomingValue.Messages {
					if err := storeIncomingMessage(ctx, dependencies, incomingMessage, incomingValue.Contacts, incomingValue.Metadata); err != nil {
						if strings.Contains(err.Error(), "duplicate") {
							continue
						}
						return err
					}
				}
				for _, status := range incomingValue.Statuses {
					if err := storeIncomingStatus(ctx, dependencies, status); err != nil {
						return err
					}
				}
			} else if change.Field == "message_template_status_update" {
				var templateStatus dto_wa.IncomingTemplateStatusChange
				if err := json.Unmarshal(change.Value, &templateStatus); err != nil {
					return fmt.Errorf("invalid message template status change value: %w", err)
				}
			}
		}
	}
	return nil
}

func storeIncomingStatus(ctx context.Context, dependencies *service.Dependencies, status dto_wa.Status) error {
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
	event := dao_wa.MessageStatus{
		WAMessageId: status.ID,
		MessageId:   message.Id,
		Status:      messageStatus,
		Timestamp:   timestamp,
		Payload:     status.Payload,
	}
	if len(status.Errors) > 0 {
		var errors []string
		for _, err := range status.Errors {
			errors = append(errors, err.Error())
		}
		event.ErrorMessage = strings.Join(errors, "\n\n")
	}
	if err := transaction.WAMessageStatusRepository().Insert(ctx, &event); err != nil {
		return err
	}
	// Persist every callback for audit, but only advance the current status. Meta may
	// deliver callbacks out of order or omit delivered when a message is read directly.
	// This prevents stale sent, delivered, or failed callbacks from regressing read/played.
	if helper.CanTransitionWAMessageStatus(message.Status, messageStatus) {
		message.Status = messageStatus
		message.WAMessageId = status.ID // ensure waMessageId is there
		message.ErrorMessage = event.ErrorMessage
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
	_ = dependencies.Ably.Publish("status", channelName, dto_wa.NewMessageStatus(event))
	return nil
}

func storeIncomingMessage(ctx context.Context, dependencies *service.Dependencies, incomingMessage dto_wa.IncomingMessage, contacts []dto_wa.IncomingContact, metadata dto_wa.IncomingMetadata) error {
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

func RetrySendingMessage(ctx context.Context, messageId int32, mqMessage *service.MessageQueueMessage, dependencies *service.Dependencies) error {
	if mqMessage == nil {
		return errors.New("message queue message is required")
	}
	acknowledge := func() {
		if mqMessage.ReceiptHandle != "" {
			if deleteErr := dependencies.MessageQueue.DeleteMessage(mqMessage.ReceiptHandle); deleteErr != nil {
				dependencies.Logger.ErrorFunction(deleteErr, mqMessage.MessageID)
			}
		}
	}
	message, err := dependencies.UnitOfWork.WAMessageRepository().GetById(ctx, messageId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			acknowledge()
			return nil
		}
		// returning error will make SQM retry automatcially
		return err
	}
	now := time.Now().UTC()
	if !shouldRetryWAMessageStatus(message.Status) || message.NextAttemptAt == nil || message.NextAttemptAt.After(now) {
		acknowledge()
		return nil
	}
	stopRetry := func(reason string) error {
		message.Status = types.WAMessageStatusFailed
		message.NextAttemptAt = nil
		message.ErrorMessage = reason
		if err := dependencies.UnitOfWork.WAMessageRepository().Update(ctx, message); err != nil {
			return err
		}
		acknowledge()
		return nil
	}
	resetNextAttemptAt := func() error {
		message.NextAttemptAt = helper.ConvertToPointer(now.Add(time.Minute))
		if err := dependencies.UnitOfWork.WAMessageRepository().Update(ctx, message); err != nil {
			return err
		}
		return nil
	}
	if message.Attempts >= 3 {
		return stopRetry("maximum send attempts reached")
	}
	// quickly update next_attempt_at to 15 min later so other worker won't pick it up
	updated, err := dependencies.UnitOfWork.WAMessageRepository().UpdateNextAttemptAt(ctx, message.Id, now.Add(10*time.Minute))
	if err != nil {
		return err
	}
	if !updated {
		acknowledge()
		return nil
	}
	if message.PhoneNumberId == 0 {
		return stopRetry("missing phone number ID")
	}
	phoneNumber, err := dependencies.UnitOfWork.WAPhoneNumberRepository().GetById(ctx, message.PhoneNumberId)
	if err != nil {
		// if phone number not found, don't retry
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return stopRetry("phone number not found")
		}
		if resetErr := resetNextAttemptAt(); resetErr != nil {
			return fmt.Errorf("reset retry schedule after phone number lookup: %w", resetErr)
		}
		return err
	}
	businessPortfolio, _, err := dependencies.UnitOfWork.WAPhoneNumberRepository().GetBusinessPortfolioAndAccountByUserId(ctx, phoneNumber.UserId)
	if err != nil {
		// if no business assets, don't retry
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return stopRetry("business portfolio not found")
		}
		if resetErr := resetNextAttemptAt(); resetErr != nil {
			return fmt.Errorf("reset retry schedule after business portfolio lookup: %w", resetErr)
		}
		return err
	}
	if businessPortfolio == nil {
		return stopRetry("business portfolio not found")
	}
	if strings.TrimSpace(businessPortfolio.AccessToken) == "" {
		return stopRetry("missing WhatsApp business portfolio access token")
	}
	response, err := dependencies.Whatsapp.SendMessage(ctx, phoneNumber.MetaPhoneNumberId, message.Payload, businessPortfolio.AccessToken)
	if err == nil && (response == nil || len(response.Messages) == 0 || strings.TrimSpace(response.Messages[0].ID) == "") {
		err = errors.New("WhatsApp did not return a message ID")
	}
	if err != nil {
		message.Attempts += 1
		message.ErrorMessage = err.Error()
		message.Status = types.WAMessageStatusRejected
		// schedule the next retry with linear backoff
		message.NextAttemptAt = helper.ConvertToPointer(now.Add(5 * time.Duration(message.Attempts) * time.Minute))
		if updateErr := dependencies.UnitOfWork.WAMessageRepository().Update(ctx, message); updateErr != nil {
			return updateErr
		}
		acknowledge()
		// if nextAttemptAt is set successfully, no need to return error for retry
		return nil
	}
	if response != nil && len(response.Messages) > 0 && strings.TrimSpace(response.Messages[0].ID) != "" {
		message.WAMessageId = strings.TrimSpace(response.Messages[0].ID)
	}
	message.Attempts += 1
	message.ErrorMessage = ""
	message.Status = types.WAMessageStatusAccepted
	message.NextAttemptAt = nil
	if err := dependencies.UnitOfWork.WAMessageRepository().Update(ctx, message); err != nil {
		return err
	}
	acknowledge()
	return nil
}

func shouldRetryWAMessageStatus(status types.WAMessageStatus) bool {
	switch status {
	case types.WAMessageStatusRejected:
		return true
	default:
		return false
	}
}
