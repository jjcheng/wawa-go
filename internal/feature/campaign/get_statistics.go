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

type GetStatistics struct {
	Id int32 `uri:"id" val:"required" description:"id of the campaign"`
}

func (getStatistics *GetStatistics) Validate() []exception.InputException {
	if getStatistics.Id <= 0 {
		return []exception.InputException{exception.NewInputException("id", "invalid campaign id")}
	}
	return nil
}

func (getStatistics GetStatistics) Handle(ctx context.Context, user *dto_account.User, dependencies *service.Dependencies) dto.Response[*map[types.WAMessageStatus]int] {
	if user == nil {
		return dto.NewFailedResponse[*map[types.WAMessageStatus]int](http.StatusForbidden, "you are not authenticated")
	}
	if inputErrors := getStatistics.Validate(); len(inputErrors) > 0 {
		return dto.NewInvalidInputResponse[*map[types.WAMessageStatus]int](inputErrors)
	}
	campaign, err := dependencies.UnitOfWork.CampaignRepository().GetById(ctx, getStatistics.Id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return dto.NewFailedResponse[*map[types.WAMessageStatus]int](http.StatusNotFound, "campaign not found")
		}
		return dto.NewFailedResponse[*map[types.WAMessageStatus]int](http.StatusInternalServerError, types.ExceptionMessageInternalServerError)
	}
	if campaign.UserId != user.Id {
		return dto.NewFailedResponse[*map[types.WAMessageStatus]int](http.StatusNotFound, "campaign not found")
	}
	statistics, err := dependencies.UnitOfWork.CampaignRecipientRepository().CountMessageStatusesByCampaignId(ctx, campaign.Id)
	if err != nil {
		return dto.NewFailedResponse[*map[types.WAMessageStatus]int](http.StatusInternalServerError, types.ExceptionMessageInternalServerError)
	}
	processedCount := 0
	for _, count := range statistics {
		processedCount += count
	}
	unprocessedCount := max(int(campaign.RecipientCount)-processedCount, 0)
	statistics["UNPROCESSED"] = unprocessedCount
	return dto.NewSuccessResponse(&statistics)
}

func (GetStatistics) APISettings() feature.APISettings {
	return feature.NewAPISettings(
		"Get WhatsApp campaign statistics",
		"Gets message status statistics for a campaign belonging to the authenticated user.",
		types.HttpRequestTypeUri,
		http.MethodGet,
		"/v1/campaigns/:id/statistics",
		true,
		true,
		types.APITagWA,
		[]feature.APIError{
			feature.NewAPIError(*exception.NewCustomException("campaign not found", http.StatusNotFound)),
			feature.NewAPIError(*exception.NewCustomException("you are not authenticated", http.StatusForbidden)),
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageInternalServerError, http.StatusInternalServerError)),
		},
	)
}
