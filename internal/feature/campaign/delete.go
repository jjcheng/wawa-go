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

type Delete struct {
	Id int32 `uri:"id" val:"required" description:"id of the campaign to delete"`
}

func (delete *Delete) Validate() []exception.InputException {
	if delete.Id <= 0 {
		return []exception.InputException{exception.NewInputException("id", "invalid campaign id")}
	}
	return nil
}

func (delete Delete) Handle(ctx context.Context, user *dto_account.User, dependencies *service.Dependencies) dto.Response[any] {
	if user == nil {
		return dto.NewFailedResponse[any](http.StatusForbidden, "you are not authenticated")
	}
	if inputErrors := delete.Validate(); len(inputErrors) > 0 {
		return dto.NewInvalidInputResponse[any](inputErrors)
	}
	campaign, err := dependencies.UnitOfWork.CampaignRepository().GetById(ctx, delete.Id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return dto.NewFailedResponse[any](http.StatusNotFound, "campaign not found")
		}
		return dto.NewFailedResponse[any](http.StatusInternalServerError, types.ExceptionMessageInternalServerError)
	}
	if campaign.UserId != user.Id {
		return dto.NewFailedResponse[any](http.StatusNotFound, "campaign not found")
	}
	if campaign.Status != types.CampaignStatusCancelled {
		return dto.NewFailedResponse[any](http.StatusBadRequest, "only cancelled campaigns can be deleted")
	}
	// delete any attachementurl
	if campaign.AttachmentURL != "" {
		dependencies.File.DeleteFile(campaign.AttachmentURL)
	}
	if err := dependencies.UnitOfWork.CampaignRepository().DeleteById(ctx, campaign.Id); err != nil {
		return dto.NewFailedResponse[any](http.StatusInternalServerError, types.ExceptionMessageInternalServerError)
	}
	return dto.NewEmptyResponse(true, http.StatusOK)
}

func (Delete) APISettings() feature.APISettings {
	return feature.NewAPISettings(
		"Delete WhatsApp campaign",
		"Deletes a cancelled campaign belonging to the authenticated user.",
		types.HttpRequestTypeUri,
		http.MethodDelete,
		"/v1/campaigns/:id",
		true,
		true,
		types.APITagWA,
		[]feature.APIError{
			feature.NewAPIError(*exception.NewCustomException("campaign not found", http.StatusNotFound)),
			feature.NewAPIError(*exception.NewCustomException("only cancelled campaigns can be deleted", http.StatusBadRequest)),
			feature.NewAPIError(*exception.NewCustomException("you are not authenticated", http.StatusForbidden)),
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageInternalServerError, http.StatusInternalServerError)),
		},
	)
}
