package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"github.com/aliyun/fc-runtime-go-sdk/fc"
	"github.com/jjcheng/wawa-go/internal/cfg"
	"github.com/jjcheng/wawa-go/internal/service"
	"github.com/jjcheng/wawa-go/internal/setup"
)

func main() {
	// set timezone to utc so no need to call .UTC() everytime
	time.Local = time.UTC
	// set log to stdout as error will be handled by loggerService
	log.SetOutput(os.Stdout)
	log.Println("starting dispatcher")
	log.Printf("environment: %s\n", cfg.Default().Site.Environment)
	logger := service.NewLogger()
	var dependencies *service.Dependencies

	// initialize
	fc.RegisterInitializerFunction(func(ctx context.Context) error {
		unitOfWork, err := setup.SetupDatabase(cfg.Default().Database.DSN(), logger)
		if err != nil {
			return fmt.Errorf("setup database: %w", err)
		}
		dependencies = setup.SetupServices(unitOfWork, logger)
		return nil
	})

	// Standard Function Compute runtime handler.
	handleEvent := func(ctx context.Context, raw []byte) error {
		rawStr := string(raw)
		logger.Infof("======== NEW TASK ========\n%s", rawStr)
		var envelope map[string]any
		if err := json.Unmarshal(raw, &envelope); err != nil {
			return err
		}
		// triggered by time
		if _, ok := envelope["triggerName"].(string); ok {
			var waitGroup sync.WaitGroup
			waitGroup.Go(func() {
				dispatchCampaigns(ctx, dependencies)
			})
			waitGroup.Go(func() {
				dispatchRetryMessages(ctx, dependencies)
			})
			waitGroup.Wait()
			return nil
		}
		return fmt.Errorf("unidentified type: %s", rawStr)
	}
	fc.Start(handleEvent)
}

func dispatchCampaigns(ctx context.Context, dependencies *service.Dependencies) {
	// find pending campaigns, skip if have error, will run again in next cycle
	pendingCampaigns, _ := dependencies.UnitOfWork.CampaignRepository().ListPendingCampaigns(ctx)
	dependencies.Logger.Infof("%d pending campaings", len(pendingCampaigns))
	if len(pendingCampaigns) == 0 {
		return
	}
	// dispatch each campaign to a worker task
	// if one user has multiple campaigns, only send the first, don't block other user's resources, it will be processed
	// in next time trigger
	var userCampaigns map[int32]int = make(map[int32]int)
	for _, pendingCampaign := range pendingCampaigns {
		if userCampaigns[pendingCampaign.UserId] > 0 {
			continue
		}
		userCampaigns[pendingCampaign.UserId] = 1
		dependencies.Logger.Infof("dispatching campaign id: %d", pendingCampaign.Id)
		// send a message to SMQ which will trigger worker function
		// ignore any error, the next cycle will do it again
		_, _ = dependencies.MessageQueue.PublishJob("start_campaign", pendingCampaign.Id, 0, service.MessageQueuePriorityHigh)
	}
}

func dispatchRetryMessages(ctx context.Context, dependencies *service.Dependencies) {
	// find messages to resend
	pendingMessages, _ := dependencies.UnitOfWork.WAMessageRepository().ListNeedResend(ctx)
	dependencies.Logger.Infof("%d pending retry messages", len(pendingMessages))
	if len(pendingMessages) == 0 {
		return
	}
	for _, pendingMessage := range pendingMessages {
		// ignore any error, the next cycle will do it again
		_, _ = dependencies.MessageQueue.PublishJob("retry_send_message", pendingMessage.Id, 0, service.MessageQueuePriorityHigh)
	}
}
