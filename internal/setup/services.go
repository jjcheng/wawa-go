package setup

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/jjcheng/wawa-go/internal/cfg"
	dao_wa "github.com/jjcheng/wawa-go/internal/dao/wa"
	dto_wa "github.com/jjcheng/wawa-go/internal/dto/wa"
	"github.com/jjcheng/wawa-go/internal/repository"
	"github.com/jjcheng/wawa-go/internal/service"
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
	whatsappService := dependencies.Whatsapp
	for {
		pollCtx, cancel := context.WithTimeout(ctx, time.Duration(cfg.Default().AliyunSMQ.PollingWaitSeconds+5)*time.Second)
		message, err := messageQueueService.ReceiveMessage(pollCtx)
		cancel()
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			logger.Warnf("queue listener receive error: %v", err)
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
		messageCtx, messageCancel := context.WithTimeout(context.Background(), 2*time.Minute)
		whatsappService.ReceiveMessage(messageQueueService, message, func(incomingMessage dto_wa.IncomingMessage, contact dto_wa.IncomingContact, metadata dto_wa.IncomingMetadata) error {
			return processIncomingWhatsAppMessage(messageCtx, dependencies, incomingMessage, contact, metadata)
		}, func(wabaID string, incoming dto_wa.Incoming) error {
			return storeWhatsAppHistoryMessage(messageCtx, dependencies, wabaID, incoming)
		})
		messageCancel()
	}
}

func storeWhatsAppHistoryMessage(ctx context.Context, dependencies *service.Dependencies, wabaID string, incoming dto_wa.Incoming) error {
	// rawPayload, err := json.Marshal(incoming)
	// if err != nil {
	// 	return err
	// }
	// return dependencies.UnitOfWork.WAHistoryMessageRepository().Insert(ctx, &dao_wa.HistoryMessage{
	// 	WABAId:           wabaID,
	// 	PhoneNumberId:    incomingMessage.PhoneNumberID,
	// 	From:             incomingMessage.From,
	// 	MessageId:        incomingMessage.ID,
	// 	MessageType:      incomingMessage.Type,
	// 	TextBody:         incomingMessage.Text.Body,
	// 	MessageTimestamp: incomingMessage.Timestamp,
	// 	RawPayload:       string(rawPayload),
	// })
	return nil
}

func processIncomingWhatsAppMessage(ctx context.Context, dependencies *service.Dependencies, incomingMessage dto_wa.IncomingMessage, contact dto_wa.IncomingContact, metadata dto_wa.IncomingMetadata) error {
	timestamp, err := strconv.ParseInt(incomingMessage.Timestamp, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid WhatsApp message timestamp %q: %w", incomingMessage.Timestamp, err)
	}
	message := dao_wa.Message{
		Sending:             false,
		PhoneNumber:         metadata.DisplayPhoneNumber,
		PhoneNumberId:       incomingMessage.PhoneNumberID,
		CustomerName:        contact.Profile.Name,
		CustomerPhoneNumber: incomingMessage.From,
		CustomerMetaUserId:  incomingMessage.FromUserID,
		MetaId:              incomingMessage.ID,
		Timestamp:           timestamp,
		Type:                incomingMessage.Type,
		Payload:             incomingMessage.Payload,
	}
	return dependencies.UnitOfWork.WAMessageRepository().Insert(ctx, &message)
}
