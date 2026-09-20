package feature_broadcast

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

// will call CreateMessage for each recipient, whether it's successful or failed, the broadcast will be marked completed
// if error occurred before completed and dequeueCount >= max dequeue count, send user notification
func Process(ctx context.Context, broadcastId int32, dequeueCount int, dependencies *service.Dependencies) (processErr error) {
	log.Printf("processing broadcast ID: %d", broadcastId)
	var broadcast *dao_customer.Broadcast
	// just return if we can't event get the broadcast
	err := try(ctx, func() error {
		var e error
		broadcast, e = dependencies.UnitOfWork.BroadcastRepository().GetById(ctx, broadcastId)
		return e
	})
	if err != nil {
		// no need to record error message since broadcast is nil
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("broadcast %d not found", broadcastId)
		}
		dependencies.Logger.ErrorFunction(err, broadcastId)
		return fmt.Errorf("failed to get broadcast %d: %w", broadcastId, err)
	}
	// only process if the status is PENIDNG or SENDING
	if broadcast.Status != types.BroadcastStatusPending && broadcast.Status != types.BroadcastStatusSending {
		log.Printf("broadcast %d is %s, skip", broadcastId, broadcast.Status)
		return nil
	}
	// take snapshot of original error messages, so we know if there are new errors
	originalErrorMessages := strings.Split(strings.TrimSpace(broadcast.ErrorMessage), "\n")
	errorMessages := append([]string(nil), originalErrorMessages...)
	wasPending := broadcast.Status == types.BroadcastStatusPending
	// set status to SENDING, next time trigger will not call this again
	if broadcast.Status != types.BroadcastStatusSending {
		broadcast.Status = types.BroadcastStatusSending
		err = try(ctx, func() error {
			return dependencies.UnitOfWork.BroadcastRepository().Update(ctx, broadcast)
		})
		if err != nil {
			dependencies.Logger.ErrorFunction(err, broadcast.Id)
			errorMessages = append(errorMessages, "failed to set broadcast status to sending")
			return fmt.Errorf("failed to set broadcast %d status to sending", broadcast.Id)
		}
	}
	// get user
	var user *dao_account.User
	err = try(ctx, func() error {
		var e error
		user, e = dependencies.UnitOfWork.AccountUserRepository().GetById(ctx, broadcast.UserId)
		return e
	})
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			dependencies.Logger.ErrorFunction(err, broadcast.UserId)
		}
		errorMessages = append(errorMessages, "failed to get user")
		return fmt.Errorf("failed to get user %d for broadcast %d: %w", broadcast.UserId, broadcast.Id, err)
	}
	userDTO := dto_account.NewUser(*user)
	defer func() {
		if len(errorMessages) == len(originalErrorMessages) { // no new error
			return
		}
		// if dequeueTime >= max dequeue time, this broadcast has failed
		if dequeueCount >= cfg.Default().AliyunSMQ.MaxDequeueCount {
			createNotification := feature_account_notification.Create{
				Title: fmt.Sprintf("Error occurred when sending your broadcast %s", broadcast.Name),
				Body:  fmt.Sprintf("We are sorry to inform you that your broadcast %s has failed to send despite %d deliveries. Please try again later. The errors are below:\n\n%s", broadcast.Name, dequeueCount, strings.Join(errorMessages, "\n")),
				Type:  types.NotificationTypeError,
				URL:   "/broadcasts",
			}
			createNotification.Handle(ctx, &userDTO, dependencies)
		}
		if err := try(ctx, func() error {
			broadcast.ErrorMessage = strings.Join(errorMessages, "\n")
			return dependencies.UnitOfWork.BroadcastRepository().Update(ctx, broadcast)
		}); err != nil {
			dependencies.Logger.ErrorFunction(err, broadcast.Id, errorMessages)
			log.Printf("failed to update broadcast %d error message: %v", broadcast.Id, err)
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
	// now only, if the original broadcast status is pending, push notification to info user it's started
	if wasPending {
		createNotification := feature_account_notification.Create{
			Type:  types.NotificationTypeInfo,
			Title: fmt.Sprintf("Your broadcast %s has started", broadcast.Name),
			Body:  fmt.Sprintf("We have started your broadcast %s, total %d recipients. Check the broadcast recipients page to see any individual messages that are failed to be sent.", broadcast.Name, broadcast.RecipientCount),
			URL:   fmt.Sprintf("/broadcasts/recipients?broadcast_id=%d", broadcastId),
		}
		_ = createNotification.Handle(ctx, &userDTO, dependencies)
	}
	// send in batches, record down the failed recipients
	var failedRecipientErrorMessages []string
	for {
		var recipients []dao_customer.BroadcastRecipient
		var totalPages int
		err = try(ctx, func() error {
			var e error
			recipients, _, totalPages, e = dependencies.UnitOfWork.BroadcastRecipientRepository().ListByBroadcastId(ctx, broadcast.Id, "", "", false, page, 50)
			return e
		})
		if err != nil {
			dependencies.Logger.ErrorFunction(err, broadcast.Id, page)
			errorMessages = append(errorMessages, fmt.Sprintf("error listing recipients page %d", page))
			return fmt.Errorf("failed to list recipients for broadcast %d page %d: %w", broadcast.Id, page, err)
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
			// only delete it if the broadcast is deleted
			createMessage := feature_wa_message.Create{
				CustomerId: recipient.CustomerId,
				Type:       feature_wa_message.MessageTypeTemplate,
				Template:   &recipient.Payload,
			}
			response := createMessage.Handle(ctx, &userDTO, dependencies)
			// it's ok if sending failed, as long as a message is created
			if response.Data == nil {
				log.Printf("failed to create message for broadcast %d recipient %d: %s", broadcast.Id, recipient.Id, response.Message)
				failedRecipientErrorMessages = append(failedRecipientErrorMessages, fmt.Sprintf("recipient %d: %s", recipient.Id, response.Message))
				continue
			}
			recipient.MessageId = &response.Data.Id
			if err := try(ctx, func() error {
				return dependencies.UnitOfWork.BroadcastRecipientRepository().Update(ctx, &recipient)
			}); err != nil {
				dependencies.Logger.ErrorFunction(err, broadcast.Id, recipient.Id, response.Data.Id)
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
		return fmt.Errorf("broadcast %d has %d recipient message creation/link failures", broadcast.Id, len(failedRecipientErrorMessages))
	}
	// complete broadcast, set status to COMPLETED so SMQ won't retry
	log.Print("all recipient messages created, broadcast is completed")
	broadcast.Status = types.BroadcastStatusCompleted
	err = try(ctx, func() error {
		return dependencies.UnitOfWork.BroadcastRepository().Update(ctx, broadcast)
	})
	if err != nil {
		dependencies.Logger.ErrorFunction(err, broadcast.Id)
		errorMessages = append(errorMessages, "failed to set broadcast status to COMPLETED")
		return fmt.Errorf("failed to set broadcast %d status to COMPLETED: %w", broadcast.Id, err)
	}
	// create notification
	createNotification := feature_account_notification.Create{
		Type:  types.NotificationTypeSuccess,
		Title: fmt.Sprintf("Your broadcast %s has completed.", broadcast.Name),
		Body:  fmt.Sprintf("Your broadcast %s has completed sending to a total %d recipients. Check the broadcast recipients page to see any individual messages that fail to send.", broadcast.Name, broadcast.RecipientCount),
		URL:   fmt.Sprintf("/broadcasts/recipients?broadcast_id=%d", broadcastId),
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
