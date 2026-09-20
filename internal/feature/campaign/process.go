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
	// no need to record the processErr if we can't event get the campaign
	err := retry(ctx, 3, 1000*time.Millisecond, func() error {
		var e error
		campaign, e = dependencies.UnitOfWork.CampaignRepository().GetById(ctx, campaignId)
		return e
	})
	if err != nil {
		dependencies.Logger.ErrorFunction(err, campaignId)
		// no need to record error message since campaign is nil
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("campaign %d not found", campaignId)
		}
		return fmt.Errorf("failed to get campaign %d: %w", campaignId, err)
	}
	// only process if the status is PENIDNG or SENDING
	if campaign.Status != types.CampaignStatusPending && campaign.Status != types.CampaignStatusSending {
		log.Printf("campaign %d is %s, skip", campaignId, campaign.Status)
		return nil
	}
	originalErrorMessages := strings.Split(strings.TrimSpace(campaign.ErrorMessage), "\n")
	errorMessages := strings.Split(strings.TrimSpace(campaign.ErrorMessage), "\n")
	// get user
	var user *dao_account.User
	err = retry(ctx, 3, 1000*time.Millisecond, func() error {
		var e error
		user, e = dependencies.UnitOfWork.AccountUserRepository().Get(ctx, campaign.UserId)
		return e
	})
	if err != nil {
		dependencies.Logger.ErrorFunction(err, campaign.UserId)
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
				Title: fmt.Sprintf("We failed to send your campaign %s", campaign.Name),
				Body:  fmt.Sprintf("We are sorry to inform you that your campaign %s has failed to send depite %d retries. Please try again later. The errors are below:\n\n%s", campaign.Name, dequeueCount, strings.Join(errorMessages, "\n")),
				Type:  types.NotificationTypeError,
				URL:   "/campaigns",
			}
			createNotification.Handle(ctx, &userDTO, dependencies)
		}
		if err := retry(ctx, 3, 1000*time.Millisecond, func() error {
			return dependencies.UnitOfWork.CampaignRepository().UpdateFields(ctx, campaign.Id, map[string]any{
				"error_message": strings.Join(errorMessages, "\n"),
			})
		}); err != nil {
			log.Printf("failed to update campaign %d error message: %v", campaign.Id, err)
		}
	}()
	// get wa assets
	var phoneNumber *dao_wa.PhoneNumber
	var businessAccount *dao_wa.BusinessAccount
	var businessPortfolio *dao_wa.BusinessPortfolio
	err = retry(ctx, 3, 1000*time.Millisecond, func() error {
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
	// set campaign status to SENDING
	campaign.Status = types.CampaignStatusSending
	err = retry(ctx, 3, 1000*time.Millisecond, func() error {
		return dependencies.UnitOfWork.CampaignRepository().Update(ctx, campaign)
	})
	if err != nil {
		dependencies.Logger.ErrorFunction(err, campaign.Id)
		errorMessages = append(errorMessages, "failed to update campaign status to SENDING")
		return fmt.Errorf("failed to update campaign %d to SENDING status: %w", campaign.Id, err)
	}
	// send in batches
	var failedRecipients []string
	for {
		var recipients []dao_customer.CampaignRecipient
		var totalPages int
		err = retry(ctx, 3, 1000*time.Millisecond, func() error {
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
				failedRecipients = append(failedRecipients, fmt.Sprintf("recipient %d: %s", recipient.Id, response.Message))
				continue
			}
			recipient.MessageId = &response.Data.Id
			if err := retry(ctx, 3, 1000*time.Millisecond, func() error {
				return dependencies.UnitOfWork.CampaignRecipientRepository().Update(ctx, &recipient)
			}); err != nil {
				dependencies.Logger.ErrorFunction(err, campaign.Id, recipient.Id, response.Data.Id)
				failedRecipients = append(failedRecipients, fmt.Sprintf("failed to link recipient %d to message id %d", recipient.Id, response.Data.Id))
			}
		}
		if page >= totalPages {
			break
		}
		page++
	}
	if len(failedRecipients) > 0 {
		for _, failedRecipient := range failedRecipients {
			errorMessages = append(errorMessages, failedRecipient)
		}
		return fmt.Errorf("campaign %d has %d recipient message creation/link failures: %s", campaign.Id, len(failedRecipients), strings.Join(failedRecipients, "; "))
	}
	// complete campaign
	log.Print("all recipient messages created (may not be sent successfully), campaign is completed")
	campaign.Status = types.CampaignStatusCompleted
	campaign.ErrorMessage = ""
	err = retry(ctx, 3, 1000*time.Millisecond, func() error {
		return dependencies.UnitOfWork.CampaignRepository().Update(ctx, campaign)
	})
	if err != nil {
		dependencies.Logger.ErrorFunction(err, campaign.Id)
		errorMessages = append(errorMessages, "failed to set campaign status to COMPLETED")
		return fmt.Errorf("failed to set campaign %d status as COMPLETED: %w", campaign.Id, err)
	}
	// create notification
	createNotification := feature_account_notification.Create{
		Type:  types.NotificationTypeInfo,
		Title: fmt.Sprintf("Your campaign %s has completed.", campaign.Name),
		Body:  fmt.Sprintf("Your campaign %s has completed sending to a total %d recipients. Check the campaign recipients page to see any individual messages that are failed to be sent.", campaign.Name, campaign.RecipientCount),
		URL:   fmt.Sprintf("/campaigns/recipients?campaign_id=%d", campaignId),
	}
	createNotification.Handle(ctx, &userDTO, dependencies)
	return nil
}

func retry(ctx context.Context, attempts int, delay time.Duration, fn func() error) error {
	var err error
	for i := 0; i < attempts; i++ {
		if i > 0 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delay * time.Duration(i)):
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
