package feature_campaign

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/jjcheng/wawa-go/internal/cfg"
	dao_account "github.com/jjcheng/wawa-go/internal/dao/account"
	dao_customer "github.com/jjcheng/wawa-go/internal/dao/customer"
	dao_wa "github.com/jjcheng/wawa-go/internal/dao/wa"
	dto_account "github.com/jjcheng/wawa-go/internal/dto/account"
	dto_wa "github.com/jjcheng/wawa-go/internal/dto/wa"
	feature_account_notification "github.com/jjcheng/wawa-go/internal/feature/account/notification"
	feature_wa_message "github.com/jjcheng/wawa-go/internal/feature/wa/message"
	"github.com/jjcheng/wawa-go/internal/service"
	"github.com/jjcheng/wawa-go/internal/types"
	"gorm.io/gorm"
)

// will call CreateMessage for each recipient, whether it's successful or failed, the campaign will be marked completed
// if error occurred before completed and dequeueCount >= max dequeue count, send user notification
func Process(ctx context.Context, campaignId int32, dequeueCount int, dependencies *service.Dependencies) (processErr error) {
	log.Printf("processing campaign ID: %d", campaignId)
	var campaign *dao_customer.Campaign
	// just return if we can't event get the campaign
	err := try(ctx, func() error {
		var e error
		campaign, e = dependencies.UnitOfWork.CampaignRepository().GetById(ctx, campaignId)
		return e
	})
	if err != nil {
		// no need to record error message since campaign is nil
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("campaign %d not found", campaignId)
		}
		dependencies.Logger.ErrorFunction(err, campaignId)
		return fmt.Errorf("failed to get campaign %d: %w", campaignId, err)
	}
	// only process if the status is PENIDNG or SENDING
	if campaign.Status != types.CampaignStatusPending && campaign.Status != types.CampaignStatusSending {
		log.Printf("campaign %d is %s, skip", campaignId, campaign.Status)
		return nil
	}
	// take snapshot of original error messages, so we know if there are new errors
	originalErrorMessages := strings.Split(strings.TrimSpace(campaign.ErrorMessage), "\n")
	errorMessages := append([]string(nil), originalErrorMessages...)
	wasPending := campaign.Status == types.CampaignStatusPending
	// set status to SENDING, next time trigger will not call this again
	if campaign.Status != types.CampaignStatusSending {
		campaign.Status = types.CampaignStatusSending
		err = try(ctx, func() error {
			return dependencies.UnitOfWork.CampaignRepository().Update(ctx, campaign)
		})
		if err != nil {
			dependencies.Logger.ErrorFunction(err, campaign.Id)
			errorMessages = append(errorMessages, "failed to set campaign status to sending")
			return fmt.Errorf("failed to set campaign %d status to sending", campaign.Id)
		}
	}
	// get user
	var user *dao_account.User
	err = try(ctx, func() error {
		var e error
		user, e = dependencies.UnitOfWork.AccountUserRepository().GetById(ctx, campaign.UserId)
		return e
	})
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			dependencies.Logger.ErrorFunction(err, campaign.UserId)
		}
		errorMessages = append(errorMessages, "failed to get user")
		return fmt.Errorf("failed to get user %d for campaign %d: %w", campaign.UserId, campaign.Id, err)
	}
	userDTO := dto_account.NewUser(*user)
	defer func() {
		if len(errorMessages) == len(originalErrorMessages) { // no new error
			return
		}
		// if dequeueTime >= max dequeue time, this campaign has failed
		if dequeueCount >= cfg.Default().AliyunSMQ.MaxDequeueCount {
			createNotification := feature_account_notification.Create{
				Title: fmt.Sprintf("Error occurred when sending your campaign %s", campaign.Name),
				Body:  fmt.Sprintf("We are sorry to inform you that your campaign %s has failed to send despite %d deliveries. Please try again later. The errors are below:\n\n%s", campaign.Name, dequeueCount, strings.Join(errorMessages, "\n")),
				Type:  types.NotificationTypeError,
				URL:   "/campaigns",
			}
			createNotification.Handle(ctx, &userDTO, dependencies)
		}
		if err := try(ctx, func() error {
			campaign.ErrorMessage = strings.Join(errorMessages, "\n")
			return dependencies.UnitOfWork.CampaignRepository().Update(ctx, campaign)
		}); err != nil {
			dependencies.Logger.ErrorFunction(err, campaign.Id, errorMessages)
			log.Printf("failed to update campaign %d error message: %v", campaign.Id, err)
		}
	}()
	// get wa assets
	var phoneNumber *dao_wa.PhoneNumber
	var businessAccount *dao_wa.BusinessAccount
	var businessPortfolio *dao_wa.BusinessPortfolio
	err = try(ctx, func() error {
		var e error
		phoneNumber, businessAccount, businessPortfolio, e = dependencies.UnitOfWork.WAPhoneNumberRepository().GetByUserId(ctx, user.Id)
		return e
	})
	if err != nil {
		dependencies.Logger.ErrorFunction(err, user.Id)
		errorMessages = append(errorMessages, "failed to get user's WhatsApp assets")
		return fmt.Errorf("failed to get WA assets for user %d: %w", user.Id, err)
	}
	phoneNumberDTO := dto_wa.NewPhoneNumber(*phoneNumber)
	businessAccountDTO := dto_wa.NewBusinessAccount(*businessAccount, businessPortfolio.MetaBusinessPortfolioId, businessPortfolio.Name)
	businessPortfolioDTO := dto_wa.NewBusinessPortfolio(*businessPortfolio, false)
	userDTO.WA = &dto_account.UserWA{
		PhoneNumber_:                 &phoneNumberDTO,
		BusinessAccount:              &businessAccountDTO,
		BusinessPortfolio:            &businessPortfolioDTO,
		BusinessPortfolioAccessToken: businessPortfolio.AccessToken,
	}
	page := 1
	// now only, if the original campaign status is pending, push notification to info user it's started
	if wasPending {
		createNotification := feature_account_notification.Create{
			Type:  types.NotificationTypeInfo,
			Title: fmt.Sprintf("Your campaign %s has started", campaign.Name),
			Body:  fmt.Sprintf("We have started your campaign %s, total %d recipients. Check the campaign recipients page to see any individual messages that are failed to be sent.", campaign.Name, campaign.RecipientCount),
			URL:   fmt.Sprintf("/campaigns/recipients?campaign_id=%d", campaignId),
		}
		_ = createNotification.Handle(ctx, &userDTO, dependencies)
	}
	// send in batches, record down the failed recipients
	var failedRecipientErrorMessages []string
	for {
		var recipients []dao_customer.CampaignRecipient
		var totalPages int
		err = try(ctx, func() error {
			var e error
			recipients, _, totalPages, e = dependencies.UnitOfWork.CampaignRecipientRepository().ListByCampaignId(ctx, campaign.Id, "", "", false, page, 50)
			return e
		})
		if err != nil {
			dependencies.Logger.ErrorFunction(err, campaign.Id, page)
			errorMessages = append(errorMessages, fmt.Sprintf("error listing recipients page %d", page))
			return fmt.Errorf("failed to list recipients for campaign %d page %d: %w", campaign.Id, page, err)
		}
		if len(recipients) == 0 {
			log.Printf("no more recipient to send")
			break
		}
		log.Printf("sending to %d recipients", len(recipients))
		for _, recipient := range recipients {
			// if have a message object, it's already processed
			if recipient.Message != nil {
				continue
			}
			// DO NOT include attachmentURL here since it's shared by many other messages
			// only delete it if the campaign is deleted
			createMessage := feature_wa_message.Create{
				CustomerId: recipient.CustomerId,
				Type:       feature_wa_message.MessageTypeTemplate,
				Template:   &recipient.Payload,
			}
			response := createMessage.Handle(ctx, &userDTO, dependencies)
			// it's ok if sending failed, as long as a message is created
			if response.Data == nil {
				log.Printf("failed to create message for campaign %d recipient %d: %s", campaign.Id, recipient.Id, response.Message)
				failedRecipientErrorMessages = append(failedRecipientErrorMessages, fmt.Sprintf("recipient %d: %s", recipient.Id, response.Message))
				continue
			}
			recipient.MessageId = &response.Data.Id
			if err := try(ctx, func() error {
				return dependencies.UnitOfWork.CampaignRecipientRepository().Update(ctx, &recipient)
			}); err != nil {
				dependencies.Logger.ErrorFunction(err, campaign.Id, recipient.Id, response.Data.Id)
				failedRecipientErrorMessages = append(failedRecipientErrorMessages, fmt.Sprintf("failed to link recipient %d to message id %d", recipient.Id, response.Data.Id))
			}
			// we don't care if the message is successfully sent or not, that's handled by another task
		}
		if page >= totalPages {
			break
		}
		page++
	}
	// add all recipient error messages to errorMessages and return, let SMQ retry
	if len(failedRecipientErrorMessages) > 0 {
		for _, failedRecipientErrorMessage := range failedRecipientErrorMessages {
			errorMessages = append(errorMessages, failedRecipientErrorMessage)
		}
		return fmt.Errorf("campaign %d has %d recipient message creation/link failures", campaign.Id, len(failedRecipientErrorMessages))
	}
	// complete campaign, set status to COMPLETED so SMQ won't retry
	log.Print("all recipient messages created, campaign is completed")
	campaign.Status = types.CampaignStatusCompleted
	err = try(ctx, func() error {
		return dependencies.UnitOfWork.CampaignRepository().Update(ctx, campaign)
	})
	if err != nil {
		dependencies.Logger.ErrorFunction(err, campaign.Id)
		errorMessages = append(errorMessages, "failed to set campaign status to COMPLETED")
		return fmt.Errorf("failed to set campaign %d status to COMPLETED: %w", campaign.Id, err)
	}
	// create notification
	createNotification := feature_account_notification.Create{
		Type:  types.NotificationTypeSuccess,
		Title: fmt.Sprintf("Your campaign %s has completed.", campaign.Name),
		Body:  fmt.Sprintf("Your campaign %s has completed sending to a total %d recipients. Check the campaign recipients page to see any individual messages that fail to send.", campaign.Name, campaign.RecipientCount),
		URL:   fmt.Sprintf("/campaigns/recipients?campaign_id=%d", campaignId),
	}
	_ = createNotification.Handle(ctx, &userDTO, dependencies)
	return nil
}

func try(ctx context.Context, fn func() error) error {
	var err error
	for i := range cfg.Default().AliyunSMQ.MaxDequeueCount {
		if i > 0 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(time.Second * time.Duration(i)):
			}
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if err = fn(); err == nil {
			return nil
		}
	}
	return err
}
