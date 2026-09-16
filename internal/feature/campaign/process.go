package feature_campaign

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	dao_account "github.com/jjcheng/wawa-go/internal/dao/account"
	dao_customer "github.com/jjcheng/wawa-go/internal/dao/customer"
	dao_wa "github.com/jjcheng/wawa-go/internal/dao/wa"
	dto_account "github.com/jjcheng/wawa-go/internal/dto/account"
	dto_wa "github.com/jjcheng/wawa-go/internal/dto/wa"
	feature_wa_message "github.com/jjcheng/wawa-go/internal/feature/wa/message"
	"github.com/jjcheng/wawa-go/internal/service"
	"github.com/jjcheng/wawa-go/internal/types"
	"gorm.io/gorm"
)

// will call CreateMessage for each recipient, whether it's successful or failed, the campaign will be marked completed
// no another event bridge will be scheduled, the retry will be at message level
func Process(ctx context.Context, campaignId int32, dependencies *service.Dependencies) (processErr error) {
	dependencies.Logger.Infof("processing campaign ID: %d", campaignId)
	var campaign *dao_customer.Campaign
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
		dependencies.Logger.Infof("campaign %d is %s, skip", campaignId, campaign.Status)
		return nil
	}
	errorMessage := strings.TrimSpace(campaign.ErrorMessage)
	recordError := func(format string, args ...any) {
		message := fmt.Sprintf(format, args...)
		if errorMessage == "" {
			errorMessage = message
			return
		}
		errorMessage += "\n" + message
	}
	defer func() {
		if processErr == nil || strings.TrimSpace(errorMessage) == strings.TrimSpace(campaign.ErrorMessage) {
			return
		}
		if err := retry(ctx, 3, 1000*time.Millisecond, func() error {
			return dependencies.UnitOfWork.CampaignRepository().UpdateFields(ctx, campaign.Id, map[string]any{
				"error_message": strings.TrimSpace(errorMessage),
			})
		}); err != nil {
			dependencies.Logger.Warnf("failed to update campaign %d error message: %v", campaign.Id, err)
		}
	}()

	// get user
	var user *dao_account.User
	err = retry(ctx, 3, 1000*time.Millisecond, func() error {
		var e error
		user, e = dependencies.UnitOfWork.AccountUserRepository().GetById(ctx, campaign.UserId)
		return e
	})
	if err != nil {
		dependencies.Logger.ErrorFunction(err, campaign.UserId)
		recordError("failed to get user %d: %s", campaign.UserId, err.Error())
		return fmt.Errorf("failed to get user %d for campaign %d: %w", campaign.UserId, campaign.Id, err)
	}
	userDTO := dto_account.NewUser(*user)
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
		recordError("failed to get WA assets for user %d: %s", user.Id, err.Error())
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
	dependencies.Logger.Infof("starting campaign %d", campaignId)
	page := 1
	// set campaign status to SENDING
	campaign.Status = types.CampaignStatusSending
	err = retry(ctx, 3, 1000*time.Millisecond, func() error {
		return dependencies.UnitOfWork.CampaignRepository().Update(ctx, campaign)
	})
	if err != nil {
		dependencies.Logger.ErrorFunction(err, campaign.Id)
		recordError("failed to update campaign status to SENDING: %s", err.Error())
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
			recordError("error listing recipients page %d: %s", page, err.Error())
			return fmt.Errorf("failed to list recipients for campaign %d page %d: %w", campaign.Id, page, err)
		}
		if len(recipients) == 0 {
			dependencies.Logger.Infoln("no more recipient to send")
			break
		}
		dependencies.Logger.Infof("sending to %d recipients", len(recipients))
		for _, recipient := range recipients {
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
				dependencies.Logger.Warnf("failed to create message for campaign %d recipient %d: %s", campaign.Id, recipient.Id, response.Message)
				failedRecipients = append(failedRecipients, fmt.Sprintf("recipient %d: %s", recipient.Id, response.Message))
				continue
			}
			recipient.MessageId = &response.Data.Id
			if err := retry(ctx, 3, 1000*time.Millisecond, func() error {
				return dependencies.UnitOfWork.CampaignRecipientRepository().Update(ctx, &recipient)
			}); err != nil {
				dependencies.Logger.ErrorFunction(err, campaign.Id, recipient.Id, response.Data.Id)
				failedRecipients = append(failedRecipients, fmt.Sprintf("recipient %d: link message %d: %s", recipient.Id, response.Data.Id, err.Error()))
			}
		}
		if page >= totalPages {
			break
		}
		page++
	}
	if len(failedRecipients) > 0 {
		recordError("%d recipient message creation/link failures: %s", len(failedRecipients), strings.Join(failedRecipients, "; "))
		// return error and let fc retry
		return fmt.Errorf("campaign %d has %d recipient message creation/link failures: %s", campaign.Id, len(failedRecipients), strings.Join(failedRecipients, "; "))
	}
	// complete campaign
	dependencies.Logger.Infoln("all recipient messages created (may not be sent successfully), campaign is completed")
	campaign.Status = types.CampaignStatusCompleted
	campaign.ErrorMessage = ""
	err = retry(ctx, 3, 1000*time.Millisecond, func() error {
		return dependencies.UnitOfWork.CampaignRepository().Update(ctx, campaign)
	})
	if err != nil {
		dependencies.Logger.ErrorFunction(err, campaign.Id)
		return fmt.Errorf("failed to set campaign %d status as COMPLETED: %w", campaign.Id, err)
	}
	// remove the event from EventBridge
	err = retry(ctx, 3, 1000*time.Millisecond, func() error {
		return dependencies.EventBridge.DeleteEvent(ctx, campaign.EventName())
	})
	if err != nil {
		dependencies.Logger.Warnf("failed to delete EventBridge schedule for campaign %d: %v", campaign.Id, err)
	}
	return nil
}

func retry(ctx context.Context, attempts int, delay time.Duration, fn func() error) error {
	var err error
	for i := 0; i < attempts; i++ {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if err = fn(); err == nil {
			return nil
		}
		if i < attempts-1 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delay * time.Duration(i+1)):
			}
		}
	}
	return err
}
