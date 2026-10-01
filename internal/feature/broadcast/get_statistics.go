package feature_broadcast

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
	Id int32 `uri:"id" val:"required" description:"id of the broadcast"`
}

func (getStatistics *GetStatistics) Validate() []exception.InputException {
	if getStatistics.Id <= 0 {
		return []exception.InputException{exception.NewInputException("id", "invalid broadcast id")}
	}
	return nil
}

func (getStatistics GetStatistics) Handle(ctx context.Context, user *dto_account.User, dependencies *service.Dependencies) dto.Response[*map[types.WAMessageStatus]int] {
	if user == nil {
		return dto.NewFailedResponse[*map[types.WAMessageStatus]int](http.StatusForbidden, types.ExceptionMessageForbidden, nil)
	}
	if inputErrors := getStatistics.Validate(); len(inputErrors) > 0 {
		return dto.NewInvalidInputResponse[*map[types.WAMessageStatus]int](inputErrors)
	}
	broadcast, err := dependencies.UnitOfWork.BroadcastRepository().GetById(ctx, getStatistics.Id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return dto.NewFailedResponse[*map[types.WAMessageStatus]int](http.StatusNotFound, "broadcast not found", nil)
		}
		return dto.NewFailedResponse[*map[types.WAMessageStatus]int](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
	}
	if broadcast.UserId != user.Id {
		return dto.NewFailedResponse[*map[types.WAMessageStatus]int](http.StatusUnauthorized, types.ExceptionMessageUnauthorized, nil)
	}
	statistics, err := dependencies.UnitOfWork.BroadcastRecipientRepository().CountMessageStatusesByBroadcastId(ctx, broadcast.Id)
	if err != nil {
		return dto.NewFailedResponse[*map[types.WAMessageStatus]int](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
	}
	processedCount := 0
	for _, count := range statistics {
		processedCount += count
	}
	unprocessedCount := max(int(broadcast.RecipientCount)-processedCount, 0)
	statistics["UNPROCESSED"] = unprocessedCount
	return dto.NewSuccessResponse(&statistics)
}

func (GetStatistics) APISettings() feature.APISettings {
	return feature.NewAPISettings(
		"Get a broadcast statistics",
		"Get message status statistics for a broadcast belonging to the authenticated user.",
		types.HttpRequestTypeUri,
		http.MethodGet,
		"/v1/broadcasts/:id/statistics",
		true,
		true,
		types.APITagBroadcast,
		[]feature.APIError{
			feature.NewAPIError(*exception.NewCustomException("broadcast not found", http.StatusNotFound)),
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageForbidden, http.StatusForbidden)),
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageInternalServerError, http.StatusInternalServerError)),
		},
	)
}
