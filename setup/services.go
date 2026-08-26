package setup

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jjcheng/wawa-go/internal/cfg"
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
		whatsappService.ReceiveMessage(messageQueueService, message, func(incomingMessage dto_wa.IncomingMessage) error {
			messageCtx, messageCancel := context.WithTimeout(context.Background(), 2*time.Minute)
			defer messageCancel()
			return processIncomingWhatsAppMessage(messageCtx, dependencies, incomingMessage)
		})
	}
}

func processIncomingWhatsAppMessage(ctx context.Context, dependencies *service.Dependencies, incomingMessage dto_wa.IncomingMessage) error {
	if strings.TrimSpace(incomingMessage.Text.Body) == "" {
		return nil
	}
	phoneNumberID := strings.TrimSpace(incomingMessage.PhoneNumberID)
	if phoneNumberID == "" {
		return fmt.Errorf("missing phone number id in incoming message")
	}
	go func() {
		dependencies.Whatsapp.StartTyping(ctx, phoneNumberID, incomingMessage.ID)
	}()
	// getUser := feature_account_user.Get{Identifier: phoneNumberID}
	// getUserResponse := getUser.Handle(ctx, nil, dependencies)
	// if !getUserResponse.Success || getUserResponse.Data == nil || getUserResponse.Data.App == nil {
	// 	return fmt.Errorf("failed to identify user and app from identifier: %s", phoneNumberID)
	// }
	_, err := dependencies.Whatsapp.SendMessage(ctx, &service.WhatsAppMessageRequest{
		PhoneNumberID: phoneNumberID,
		To:            incomingMessage.From,
		Type:          service.WhatsAppMessageTypeText,
		Text:          &service.WhatsAppTextObject{Body: "hello"},
	})
	return err
}
