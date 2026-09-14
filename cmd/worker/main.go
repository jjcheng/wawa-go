package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/aliyun/fc-runtime-go-sdk/fc"
	"github.com/jjcheng/wawa-go/internal/cfg"
	dto_account "github.com/jjcheng/wawa-go/internal/dto/account"
	dto_wa "github.com/jjcheng/wawa-go/internal/dto/wa"
	feature_wa_message "github.com/jjcheng/wawa-go/internal/feature/wa/message"
	"github.com/jjcheng/wawa-go/internal/service"
	"github.com/jjcheng/wawa-go/internal/setup"
	"github.com/jjcheng/wawa-go/internal/types"
)

type EventBridgeEvent struct {
	ID              string         `json:"id"`
	Source          string         `json:"source"`
	Type            string         `json:"type"`
	Subject         string         `json:"subject"`
	Time            string         `json:"time"`
	EventSourceName string         `json:"eventsourcename"`
	Task            string         `json:"task"` // campaign
	Data            map[string]any `json:"data"`
}

func main() {
	// set timezone to utc so no need to call .UTC() everytime
	time.Local = time.UTC
	// set log to stdout as error will be handled by loggerService
	log.SetOutput(os.Stdout)
	log.Println("starting worker")
	log.Printf("environment: %s\n", cfg.Default().Site.Environment)

	// setup database
	logger := service.NewLogger()
	unitOfWork, err := setup.SetupDatabase(cfg.Default().Database.DSN(), logger)
	if err != nil {
		log.Fatalf("failed to setup database: %v", err)
	}
	dependencies := setup.SetupServices(unitOfWork, logger)

	// If CLI arguments are provided, run the task directly (useful for local CLI testing)
	if len(os.Args) > 1 && os.Args[1] != "" {
		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer stop()
		runCLITask(ctx, os.Args[1:], dependencies)
		return
	}

	// Standard Function Compute Go Runtime Handler
	handleEvent := func(ctx context.Context, event EventBridgeEvent) (string, error) {
		if event.Task == "campaign" {
			var campaignId int32
			id, ok := event.Data["id"]
			if !ok {
				err := errors.New("id not found in event data")
				dependencies.Logger.Error(err)
				return "", err
			}
			switch v := id.(type) {
			case float64:
				campaignId = int32(v)
			case int:
				campaignId = int32(v)
			case int32:
				campaignId = v
			case string:
				parsed, err := strconv.Atoi(v)
				if err != nil {
					return "", fmt.Errorf("invalid id: %v", id)
				}
				campaignId = int32(parsed)
			default:
				return "", fmt.Errorf("id is not a valid number: %v", id)
			}
			processCampaign(ctx, campaignId, dependencies)
		}
		return "", nil
	}
	fc.Start(handleEvent)
}

func runCLITask(ctx context.Context, args []string, dependencies *service.Dependencies) {
	log.Println("running CLI worker task:", args[0])
	switch args[0] {
	case "campaign":
		if len(args) < 2 || args[1] == "" {
			log.Println("missing campaign ID argument")
			return
		}
		id, err := strconv.Atoi(args[1])
		if err != nil || id <= 0 {
			dependencies.Logger.Warnf("invalid campaign ID: %s\n", args[1])
			return
		}
		processCampaign(ctx, int32(id), dependencies)
	default:
		dependencies.Logger.Warnf("unknown worker task: %s\n", args[0])
	}
}

// will call CreateMessage for each recipient, whether it's successful or failed, the campaign will be marked completed
// no another event bridge will be scheduled, the retry will be at message level
func processCampaign(ctx context.Context, campaignId int32, dependencies *service.Dependencies) {
	dependencies.Logger.Infof("processing campaign ID: %d\n", campaignId)
	campaign, err := dependencies.UnitOfWork.CampaignRepository().GetById(ctx, campaignId)
	if err != nil {
		dependencies.Logger.ErrorFunction(err, campaignId)
		return
	}
	if campaign.Status != types.CampaignStatusPending {
		dependencies.Logger.Infof("campaign %d is not pending (%s), skip\n", campaignId, campaign.Status)
		return
	}
	// get user
	user, err := dependencies.UnitOfWork.AccountUserRepository().GetById(ctx, campaign.UserId)
	if err != nil {
		dependencies.Logger.ErrorFunction(err, campaign.UserId)
		return
	}
	userDTO := dto_account.NewUser(*user)
	// get wa assets
	phoneNumber, businessAccount, businessPortfolio, err := dependencies.UnitOfWork.WAPhoneNumberRepository().GetByUserId(ctx, user.Id)
	if err != nil {
		dependencies.Logger.ErrorFunction(err, user.Id)
		return
	}
	phoneNumberDTO := dto_wa.NewPhoneNumber(*phoneNumber)
	phoneNumberDTO.UserName = userDTO.Name
	businessAccountDTO := dto_wa.NewBusinessAccount(*businessAccount, businessPortfolio.MetaBusinessPortfolioId, businessPortfolio.Name)
	businessPortfolioDTO := dto_wa.NewBusinessPortfolio(*businessPortfolio, false)
	userDTO.WA = &dto_account.UserWA{
		PhoneNumber_:                 &phoneNumberDTO,
		BusinessAccount:              &businessAccountDTO,
		BusinessPortfolio:            &businessPortfolioDTO,
		BusinessPortfolioAccessToken: businessPortfolio.AccessToken,
	}
	dependencies.Logger.Infof("starting campaign %d\n", campaignId)
	page := 1
	// send in batches
	for {
		recipients, _, totalPages, err := dependencies.UnitOfWork.CampaignRecipientRepository().ListByCampaignId(ctx, campaign.Id, "", "", page, 50)
		if err != nil {
			dependencies.Logger.ErrorFunction(err, campaign.Id, page)
			campaign.Status = types.CampaignStatusPending
			if updateErr := dependencies.UnitOfWork.CampaignRepository().Update(ctx, campaign); updateErr != nil {
				dependencies.Logger.ErrorFunction(updateErr, campaign.Id)
			}
			return
		}
		if len(recipients) == 0 {
			dependencies.Logger.Infoln("no more campaign recipients")
			break
		}
		// update campaign status only here
		if campaign.Status != types.CampaignStatusSending {
			// set campaign status to pending
			campaign.Status = types.CampaignStatusSending
			if err := dependencies.UnitOfWork.CampaignRepository().Update(ctx, campaign); err != nil {
				dependencies.Logger.ErrorFunction(err, campaign.Id)
				return
			}
		}
		dependencies.Logger.Infof("sending to %d recipients\n", len(recipients))
		for _, recipient := range recipients {
			// skip if status is not pending, this by right should not happen
			if recipient.Status != types.CampaignRecipientStatusPending {
				continue
			}
			// attachmentUrl is not set here becuase it's shared among all recipients
			createMessage := feature_wa_message.Create{
				CustomerId:          recipient.CustomerId,
				Type:                feature_wa_message.MessageTypeTemplate,
				Template:            &recipient.Payload,
				CampaignRecipientId: &recipient.Id,
			}
			// we don't care about the results, let it handle the status itself
			_ = createMessage.Handle(ctx, &userDTO, dependencies)
			// mark it as complete
			recipient.Status = types.CampaignRecipientStatusCompleted
			if err := dependencies.UnitOfWork.CampaignRecipientRepository().Update(ctx, &recipient); err != nil {
				dependencies.Logger.ErrorFunction(err, recipient.Id)
			}
		}
		if page >= totalPages {
			break
		}
		page++
	}
	// complete campaign
	dependencies.Logger.Infoln("all recipient messages created (may not be sent successfully), campaign is completed")
	campaign.Status = types.CampaignStatusCompleted
	if err := dependencies.UnitOfWork.CampaignRepository().Update(ctx, campaign); err != nil {
		dependencies.Logger.ErrorFunction(err, campaign.Id)
	}
	// remove the event from EventBridge
	if err := dependencies.EventBridge.DeleveEvent(ctx, campaign.EventName()); err != nil {
		dependencies.Logger.Warnf("failed to delete EventBridge schedule for campaign %d: %v\n", campaign.Id, err)
	}
}
