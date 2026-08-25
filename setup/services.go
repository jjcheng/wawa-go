package setup

import (
	"context"
	"time"

	"github.com/jjcheng/wawa-go/internal/cfg"
	dto_wa "github.com/jjcheng/wawa-go/internal/dto/wa"
	"github.com/jjcheng/wawa-go/internal/repository"
	"github.com/jjcheng/wawa-go/internal/service"
)

// initializes and returns all application services
func SetupServices(unitOfWork repository.UnitOfWork, logger *service.Logger, startMessageQueue bool) *service.Dependencies {
	fileService := service.NewFileService(logger)
	messageQueueService := service.NewMessageQueue(logger)
	whatsappService := service.NewWhatsapp(logger)
	dependencies := service.NewDependencies(unitOfWork, logger, fileService, messageQueueService, whatsappService)
	if cfg.Default().AliyunSMQ.Listening && startMessageQueue {
		go startQueueListener(dependencies)
	}
	return dependencies
}

func startQueueListener(dependencies *service.Dependencies) {
	logger := dependencies.Logger
	messageQueueService := dependencies.MessageQueue
	whatsappService := dependencies.Whatsapp
	for {
		ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cfg.Default().AliyunSMQ.PollingWaitSeconds+5)*time.Second)
		message, err := messageQueueService.ReceiveMessage(ctx)
		cancel()
		if err != nil {
			logger.Warnf("queue listener receive error: %v", err)
			time.Sleep(time.Second)
			continue
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
	// if strings.TrimSpace(incomingMessage.Text.Body) == "" {
	// 	return nil
	// }
	// phoneNumberID := strings.TrimSpace(incomingMessage.PhoneNumberID)
	// if phoneNumberID == "" {
	// 	return fmt.Errorf("missing phone number id in incoming message")
	// }
	// go func() {
	// 	dependencies.Whatsapp.StartTyping(ctx, phoneNumberID, incomingMessage.ID)
	// }()
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
	//return err
	return nil
}
