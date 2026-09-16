package setup

import (
	"github.com/jjcheng/wawa-go/internal/repository"
	"github.com/jjcheng/wawa-go/internal/service"
)

// const webhookMessageVisibilityTimeoutSeconds = 180

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

// func StartQueueListener(ctx context.Context, dependencies *service.Dependencies) {
// 	for {
// 		pollCtx, cancel := context.WithTimeout(ctx, time.Duration(cfg.Default().AliyunSMQ.PollingWaitSeconds+25)*time.Second)
// 		message, err := dependencies.MessageQueue.ReceiveMessage(pollCtx)
// 		cancel()
// 		if err != nil {
// 			if ctx.Err() != nil {
// 				return
// 			}
// 			if errors.Is(err, context.DeadlineExceeded) {
// 				continue
// 			}
// 			dependencies.Logger.ErrorFunction(err)
// 			select {
// 			case <-ctx.Done():
// 				return
// 			case <-time.After(time.Second):
// 			}
// 			continue
// 		}
// 		if ctx.Err() != nil {
// 			return
// 		}
// 		if message == nil {
// 			continue
// 		}
// 		if err := dependencies.MessageQueue.ExtendMessageVisibility(message, webhookMessageVisibilityTimeoutSeconds); err != nil {
// 			dependencies.Logger.ErrorFunction(err, message.MessageID)
// 			continue
// 		}
// 		body := []byte(strings.TrimSpace(message.Body))
// 		if decodedBody, err := base64.StdEncoding.DecodeString(message.Body); err == nil && json.Valid(decodedBody) {
// 			body = decodedBody
// 		}
// 		queueJob, err := helper.DeserializeJSON[service.QueueJob](string(body))
// 		if err != nil {
// 			dependencies.Logger.ErrorFunction(err, message.MessageID)
// 			continue
// 		}
// 		if queueJob.Type == "wa_incoming" {
// 			messageCtx, messageCancel := context.WithTimeout(context.Background(), 2*time.Minute)
// 			incoming, err := helper.DeserializeJSON[dto_wa.Incoming](string(queueJob.Data))
// 			if err != nil {
// 				messageCancel()
// 				dependencies.Logger.ErrorFunction(err, message.MessageID)
// 				continue
// 			}
// 			ex := processWAIncoming(messageCtx, dependencies, *incoming)
// 			messageCancel()
// 			if ex != nil {
// 				// delete the queued message if already stored
// 				if strings.Contains(ex.Message, "duplicate key value violates") {
// 					if deleteErr := dependencies.MessageQueue.DeleteMessage(message.ReceiptHandle); deleteErr != nil {
// 						dependencies.Logger.ErrorFunction(deleteErr, message.MessageID)
// 					}
// 				}
// 				continue
// 			}
// 			if err := dependencies.MessageQueue.DeleteMessage(message.ReceiptHandle); err != nil {
// 				dependencies.Logger.ErrorFunction(err, message.MessageID)
// 			}
// 		}
