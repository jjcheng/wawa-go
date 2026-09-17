package main

import (
	"context"
	"errors"
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
	var dependencies *service.Dependencies

	// initialize
	fc.RegisterInitializerFunction(func(ctx context.Context) {
		// set timezone to utc so no need to call .UTC() everytime
		time.Local = time.UTC
		// set log to stdout as error will be handled by loggerService
		log.SetOutput(os.Stdout)
		log.Printf("======== dispatcher function invoked (%s) ========", cfg.Default().Site.Environment)
		logger := service.NewLogger()
		unitOfWork, err := setup.SetupDatabase(cfg.Default().Database.DSN(), logger)
		if err != nil {
			return
		}
		dependencies = setup.SetupServices(unitOfWork, logger)
	})

	// Standard Function Compute runtime handler, dispatcher function can only be triggered by time-trigger every x min
	fc.Start(func(ctx context.Context, raw []byte) error {
		if dependencies == nil {
			return errors.New("dispatcher dependencies are not initialized; configure the Function Compute initializer")
		}
		var waitGroup sync.WaitGroup
		waitGroup.Go(func() {
			dispatchCampaigns(ctx, dependencies)
		})
		waitGroup.Go(func() {
			dispatchRetryMessages(ctx, dependencies)
		})
		waitGroup.Wait()
		return nil
	})

	fc.RegisterPreStopFunction(func(ctx context.Context) {
		log.Print("======== dispatcher function ended ========")
	})
}

func dispatchCampaigns(ctx context.Context, dependencies *service.Dependencies) {
	// find pending campaigns, skip if have error, will run again in next cycle
	pendingCampaigns, _ := dependencies.UnitOfWork.CampaignRepository().ListPendingCampaigns(ctx)
	log.Printf("%d pending campaings", len(pendingCampaigns))
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
		log.Printf("dispatching campaign id: %d", pendingCampaign.Id)
		// send a message to SMQ which will trigger worker function
		// ignore any error, the next cycle will do it again
		_, _ = dependencies.MessageQueue.PublishJob("start_campaign", pendingCampaign.Id, 0, service.MessageQueuePriorityHigh)
	}
}

func dispatchRetryMessages(ctx context.Context, dependencies *service.Dependencies) {
	// find messages to resend
	pendingMessages, _ := dependencies.UnitOfWork.WAMessageRepository().ListNeedResend(ctx)
	log.Printf("%d pending retry messages", len(pendingMessages))
	if len(pendingMessages) == 0 {
		return
	}
	for _, pendingMessage := range pendingMessages {
		// ignore any error, the next cycle will do it again
		_, _ = dependencies.MessageQueue.PublishJob("retry_send_message", pendingMessage.Id, 0, service.MessageQueuePriorityHigh)
	}
}
