package setup

import (
	"context"
	"encoding/json"
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
		whatsappService.ReceiveMessage(messageQueueService, message, func(incomingMessage dto_wa.IncomingMessage) error {
			return processIncomingWhatsAppMessage(messageCtx, dependencies, incomingMessage)
		}, func(wabaID string, incomingMessage dto_wa.IncomingMessage) error {
			return storeWhatsAppHistoryMessage(messageCtx, dependencies, wabaID, incomingMessage)
		})
		messageCancel()
	}
}

func storeWhatsAppHistoryMessage(ctx context.Context, dependencies *service.Dependencies, wabaID string, incomingMessage dto_wa.IncomingMessage) error {
	rawPayload, err := json.Marshal(incomingMessage)
	if err != nil {
		return err
	}
	return dependencies.UnitOfWork.WAHistoryMessageRepository().Insert(ctx, &dao_wa.HistoryMessage{
		WABAId:           wabaID,
		PhoneNumberId:    incomingMessage.PhoneNumberID,
		From:             incomingMessage.From,
		MessageId:        incomingMessage.ID,
		MessageType:      incomingMessage.Type,
		TextBody:         incomingMessage.Text.Body,
		MessageTimestamp: incomingMessage.Timestamp,
		RawPayload:       string(rawPayload),
	})
}

func processIncomingWhatsAppMessage(ctx context.Context, dependencies *service.Dependencies, incomingMessage dto_wa.IncomingMessage) error {
	go func() {
		businessPortfolio, ex := dependencies.UnitOfWork.WAPhoneNumberRepository().GetBusinessPortfolioByMetaPhoneNumberId(ctx, incomingMessage.PhoneNumberID)
		if ex != nil {
			return
		}
		dependencies.Whatsapp.StartTyping(ctx, incomingMessage.PhoneNumberID, incomingMessage.ID, businessPortfolio.AccessToken)
	}()
	// getUser := feature_account_user.Get{Identifier: phoneNumberID}
	// getUserResponse := getUser.Handle(ctx, nil, dependencies)
	// if !getUserResponse.Success || getUserResponse.Data == nil || getUserResponse.Data.App == nil {
	// 	return fmt.Errorf("failed to identify user and app from identifier: %s", phoneNumberID)
	// }
	// _, err := dependencies.Whatsapp.SendMessage(ctx, &service.WhatsAppMessageRequest{
	// 	PhoneNumberID: phoneNumberID,
	// 	To:            incomingMessage.From,
	// 	Type:          service.WhatsAppMessageTypeText,
	// 	Text:          &service.WhatsAppTextObject{Body: "hello"},
	// })
	return nil
}
