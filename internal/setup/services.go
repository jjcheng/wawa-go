package setup

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/jjcheng/wawa-go/internal/cfg"
	dao_wa "github.com/jjcheng/wawa-go/internal/dao/wa"
	dto_wa "github.com/jjcheng/wawa-go/internal/dto/wa"
	"github.com/jjcheng/wawa-go/internal/helper"
	"github.com/jjcheng/wawa-go/internal/repository"
	"github.com/jjcheng/wawa-go/internal/service"
	"github.com/jjcheng/wawa-go/internal/types"
)

// initializes and returns all application services
func SetupServices(unitOfWork repository.UnitOfWork, logger *service.Logger) *service.Dependencies {
	fileService := service.NewFileService(logger)
	messageQueueService := service.NewMessageQueue(logger)
	whatsappService := service.NewWhatsapp(logger)
	dependencies := service.NewDependencies(unitOfWork, logger, fileService, messageQueueService, whatsappService)
	return dependencies
}

func StartQueueListener(ctx context.Context, dependencies *service.Dependencies) {
	logger := dependencies.Logger
	messageQueueService := dependencies.MessageQueue
	for {
		pollCtx, cancel := context.WithTimeout(ctx, time.Duration(cfg.Default().AliyunSMQ.PollingWaitSeconds+25)*time.Second)
		message, err := messageQueueService.ReceiveMessage(pollCtx)
		cancel()
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			logger.ErrorFunction(err)
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
		body := []byte(strings.TrimSpace(message.Body))
		if decodedBody, err := base64.StdEncoding.DecodeString(message.Body); err == nil && json.Valid(decodedBody) {
			body = decodedBody
		}
		incoming, err := helper.DeserializeJSON[dto_wa.Incoming](string(body))
		if err != nil {
			logger.ErrorFunction(err, message.MessageID)
			continue
		}
		messageCtx, messageCancel := context.WithTimeout(context.Background(), 2*time.Minute)
		err = processWAIncoming(messageCtx, dependencies, *incoming)
		messageCancel()
		if err != nil {
			// delete the queued message if already stored
			if strings.Contains(err.Error(), "duplicate key value violates") {
				if err := messageQueueService.DeleteMessage(message.ReceiptHandle); err != nil {
					logger.ErrorFunction(err, message.MessageID)
				}
			}
			continue
		}
		if err := messageQueueService.DeleteMessage(message.ReceiptHandle); err != nil {
			logger.ErrorFunction(err, message.MessageID)
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
	event := dao_wa.MessageStatusEvent{
		WAMessageId: status.ID,
		Status:      messageStatus,
		Timestamp:   timestamp,
		Payload:     status.Payload,
	}
	if err := transaction.WAMessageStatusEventRepository().Insert(ctx, &event); err != nil {
		return err
	}
	if err := transaction.WAMessageRepository().UpdateStatusByWAMessageId(ctx, status.ID, messageStatus); err != nil {
		return err
	}
	if err := transaction.CommitTransaction(); err != nil {
		return err
	}
	committed = true
	return nil
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
	message := dao_wa.Message{
		Sending:             false,
		PhoneNumber:         metadata.DisplayPhoneNumber,
		PhoneNumberId:       metadata.PhoneNumberID,
		CustomerName:        contact.Profile.Name,
		CustomerPhoneNumber: incomingMessage.From,
		CustomerMetaUserId:  incomingMessage.FromUserID,
		WAMessageId:         incomingMessage.ID,
		Timestamp:           timestamp,
		Type:                incomingMessage.Type,
		Payload:             incomingMessage.Payload,
	}
	if err := dependencies.UnitOfWork.WAMessageRepository().Insert(ctx, &message); err != nil {
		return err
	}
	return nil
}
