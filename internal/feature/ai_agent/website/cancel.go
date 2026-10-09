package feature_ai_agent_website

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/jjcheng/wawa-go/internal/dto"
	dto_account "github.com/jjcheng/wawa-go/internal/dto/account"
	"github.com/jjcheng/wawa-go/internal/exception"
	"github.com/jjcheng/wawa-go/internal/feature"
	"github.com/jjcheng/wawa-go/internal/service"
	"github.com/jjcheng/wawa-go/internal/types"
	"gorm.io/gorm"
)

type Cancel struct {
	Id int32 `uri:"id" val:"required" description:"id of the website"`
}

func (cancel *Cancel) Validate() []exception.InputException {
	if cancel.Id <= 0 {
		return []exception.InputException{exception.NewInputException("id", "invalid website id")}
	}
	return nil
}

func (cancel Cancel) Handle(ctx context.Context, user *dto_account.User, dependencies *service.Dependencies) dto.Response[any] {
	if user == nil {
		return dto.NewFailedResponse[any](http.StatusForbidden, types.ExceptionMessageForbidden, nil)
	}
	if user.Type != types.UserTypeMaster {
		return dto.NewFailedResponse[any](http.StatusUnauthorized, types.ExceptionMessageUnauthorized, nil)
	}
	if inputErrors := cancel.Validate(); len(inputErrors) > 0 {
		return dto.NewInvalidInputResponse[any](inputErrors)
	}
	website, err := dependencies.UnitOfWork.AIAgentWebsiteRepository().GetById(ctx, cancel.Id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return dto.NewFailedResponse[any](http.StatusNotFound, "website not found", nil)
		}
		return dto.NewFailedResponse[any](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
	}
	if website.BusinessAccountId != user.BusinessAccountId {
		return dto.NewFailedResponse[any](http.StatusUnauthorized, types.ExceptionMessageUnauthorized, nil)
	}
	if strings.TrimSpace(website.JobId) == "" {
		return dto.NewFailedResponse[any](http.StatusConflict, "website has no crawl job", fmt.Errorf("website %d has no crawl job ID", cancel.Id))
	}
	if err := dependencies.Cloudflare.CancelCrawl(ctx, website.JobId); err != nil {
		return dto.NewFailedResponse[any](http.StatusBadGateway, types.ExceptionMessageBadGateway, err)
	}
	if err := dependencies.UnitOfWork.AIAgentWebsiteRepository().UpdateFields(ctx, cancel.Id, map[string]any{"status": "Cancelled"}); err != nil {
		return dto.NewFailedResponse[any](http.StatusInternalServerError, types.ExceptionMessageInternalServerError,
			fmt.Errorf("crawl job %s cancelled but website %d status update failed: %w", website.JobId, cancel.Id, err))
	}
	return dto.NewEmptyResponse(true, http.StatusOK)
}

func (Cancel) APISettings() feature.APISettings {
	return feature.NewAPISettings(
		"Cancel AI agent website crawl",
		"Cancels the Cloudflare crawl job for a website in the business account without deleting the website.",
		types.HttpRequestTypeUri,
		http.MethodPost,
		"/v1/ai-agent/websites/:id/cancel",
		true,
		true,
		types.APITagAIAgent,
		[]feature.APIError{
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageForbidden, http.StatusForbidden)),
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageUnauthorized, http.StatusUnauthorized)),
			feature.NewAPIError(*exception.NewCustomException("website not found", http.StatusNotFound)),
			feature.NewAPIError(*exception.NewCustomException("website has no crawl job", http.StatusConflict)),
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageBadGateway, http.StatusBadGateway)),
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageInternalServerError, http.StatusInternalServerError)),
		},
	)
}
