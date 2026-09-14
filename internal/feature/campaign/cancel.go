package feature_campaign

import (
	"context"
	"errors"
	"net/http"

	"github.com/jjcheng/wawa-go/internal/dto"
	dto_account "github.com/jjcheng/wawa-go/internal/dto/account"
	"github.com/jjcheng/wawa-go/internal/exception"
	"github.com/jjcheng/wawa-go/internal/feature"
	"github.com/jjcheng/wawa-go/internal/service"
	"github.com/jjcheng/wawa-go/internal/types"
	"gorm.io/gorm"
)

type Cancel struct {
	Id int32 `uri:"id" val:"required" description:"id of the campaign to cancel"`
}

func (cancel *Cancel) Validate() []exception.InputException {
	if cancel.Id <= 0 {
		return []exception.InputException{exception.NewInputException("id", "invalid campaign id")}
	}
	return nil
}

func (cancel Cancel) Handle(ctx context.Context, user *dto_account.User, dependencies *service.Dependencies) dto.Response[any] {
	if user == nil {
		return dto.NewFailedResponse[any](http.StatusForbidden, "you are not authenticated")
	}
	if errors := cancel.Validate(); len(errors) > 0 {
		return dto.NewInvalidInputResponse[any](errors)
	}
	campaign, err := dependencies.UnitOfWork.CampaignRepository().GetById(ctx, cancel.Id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return dto.NewFailedResponse[any](http.StatusNotFound, "campaign not found")
		}
		return dto.NewFailedResponse[any](http.StatusInternalServerError, types.ExceptionMessageInternalServerError)
	}
	if campaign.UserId != user.Id {
		return dto.NewFailedResponse[any](http.StatusNotFound, "campaign not found")
	}
	if campaign.Status != types.CampaignStatusPending {
		return dto.NewFailedResponse[any](http.StatusBadRequest, "only pending campaigns can be cancelled")
	}
	// start a transaction to also cancel all campaign_recipients
	transaction := dependencies.UnitOfWork.BeginTransaction()
	committed := false
	defer func() {
		if !committed {
			transaction.Rollback()
		}
	}()
	if err := transaction.CampaignRepository().UpdateFields(ctx, campaign.Id, map[string]any{
		"status": types.CampaignStatusCancelled,
	}); err != nil {
		return dto.NewFailedResponse[any](http.StatusInternalServerError, types.ExceptionMessageInternalServerError)
	}
	if err := transaction.CampaignRecipientRepository().CancelByCampaignId(ctx, campaign.Id); err != nil {
		return dto.NewFailedResponse[any](http.StatusInternalServerError, types.ExceptionMessageInternalServerError)
	}
	if err := transaction.CommitTransaction(); err != nil {
		return dto.NewFailedResponse[any](http.StatusInternalServerError, types.ExceptionMessageInternalServerError)
	}
	committed = true
	if err := dependencies.EventBridge.DeleveEvent(ctx, campaign.EventName()); err != nil {
		dependencies.Logger.Warnf("failed to delete EventBridge schedule for campaign %d: %v", campaign.Id, err)
	}
	return dto.NewEmptyResponse(true, http.StatusOK)
}

func (Cancel) APISettings() feature.APISettings {
	return feature.NewAPISettings(
		"Cancel WhatsApp campaign",
		"Cancels a pending campaign belonging to the authenticated user.",
		types.HttpRequestTypeUri,
		http.MethodPatch,
		"/v1/campaigns/:id/cancel",
		true,
		true,
		types.APITagWA,
		[]feature.APIError{
			feature.NewAPIError(*exception.NewCustomException("campaign not found", http.StatusNotFound)),
			feature.NewAPIError(*exception.NewCustomException("only pending campaigns can be cancelled", http.StatusBadRequest)),
			feature.NewAPIError(*exception.NewCustomException("you are not authenticated", http.StatusForbidden)),
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageInternalServerError, http.StatusInternalServerError)),
		},
	)
}
