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

type Archive struct {
	Id       int32 `uri:"id" val:"required" description:"id of the campaign"`
	Archived bool  `form:"archived" val:"required" description:"archived or not"`
}

func (archive *Archive) Validate() []exception.InputException {
	if archive.Id <= 0 {
		return []exception.InputException{exception.NewInputException("id", "invalid campaign id")}
	}
	return nil
}

func (archive Archive) Handle(ctx context.Context, user *dto_account.User, dependencies *service.Dependencies) dto.Response[any] {
	if user == nil {
		return dto.NewFailedResponse[any](http.StatusForbidden, "you are not authenticated")
	}
	if errors := archive.Validate(); len(errors) > 0 {
		return dto.NewInvalidInputResponse[any](errors)
	}
	campaign, err := dependencies.UnitOfWork.WACampaignRepository().GetById(ctx, archive.Id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return dto.NewFailedResponse[any](http.StatusNotFound, "campaign not found")
		}
		return dto.NewFailedResponse[any](http.StatusInternalServerError, types.ExceptionMessageInternalServerError)
	}
	if campaign.UserId != user.Id {
		return dto.NewFailedResponse[any](http.StatusNotFound, "campaign not found")
	}
	// status must not be pending
	if campaign.Status == types.CampaignStatusPending || campaign.Status == types.CampaignStatusProcessing {
		return dto.NewFailedResponse[any](http.StatusBadRequest, "campaign status cannot be pending or processing")
	}
	if err := dependencies.UnitOfWork.WACampaignRepository().UpdateFields(ctx, campaign.Id, map[string]any{"archived": archive}); err != nil {
		return dto.NewFailedResponse[any](http.StatusInternalServerError, types.ExceptionMessageInternalServerError)
	}
	return dto.NewEmptyResponse(true, http.StatusOK)
}

func (Archive) APISettings() feature.APISettings {
	return feature.NewAPISettings(
		"Archive WhatsApp campaign",
		"Archive or unarchive a campaign belonging to the authenticated user.",
		types.HttpRequestTypeUri,
		http.MethodPatch,
		"/v1/campaigns/:id/archive",
		true,
		true,
		types.APITagWA,
		[]feature.APIError{
			feature.NewAPIError(*exception.NewCustomException("campaign not found", http.StatusNotFound)),
			feature.NewAPIError(*exception.NewCustomException("you are not authenticated", http.StatusForbidden)),
			feature.NewAPIError(*exception.NewCustomException("campaign status cannot be pending or processing", http.StatusBadRequest)),
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageInternalServerError, http.StatusInternalServerError)),
		},
	)
}
