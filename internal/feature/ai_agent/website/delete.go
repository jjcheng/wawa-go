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

type Delete struct {
	Id int32 `uri:"id" val:"required" description:"id of the website"`
}

func (delete *Delete) Validate() []exception.InputException {
	if delete.Id <= 0 {
		return []exception.InputException{exception.NewInputException("id", "invalid website id")}
	}
	return nil
}

func (delete Delete) Handle(ctx context.Context, user *dto_account.User, dependencies *service.Dependencies) dto.Response[any] {
	if user == nil {
		return dto.NewFailedResponse[any](http.StatusForbidden, types.ExceptionMessageForbidden, nil)
	}
	if user.Type != types.UserTypeMaster {
		return dto.NewFailedResponse[any](http.StatusUnauthorized, types.ExceptionMessageUnauthorized, nil)
	}
	if inputErrors := delete.Validate(); len(inputErrors) > 0 {
		return dto.NewInvalidInputResponse[any](inputErrors)
	}
	website, err := dependencies.UnitOfWork.AIAgentWebsiteRepository().GetById(ctx, delete.Id)
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
		return dto.NewFailedResponse[any](http.StatusConflict, "website has no crawl job", fmt.Errorf("website %d has no crawl job ID", delete.Id))
	}
	crawl, err := dependencies.Cloudflare.GetCrawlStatus(ctx, website.JobId, 0, 1)
	if err != nil {
		return dto.NewFailedResponse[any](http.StatusBadGateway, types.ExceptionMessageBadGateway, err)
	}
	if !crawl.Success {
		return dto.NewFailedResponse[any](http.StatusBadGateway, types.ExceptionMessageBadGateway, fmt.Errorf("Cloudflare failed to return crawl status for job %s", website.JobId))
	}
	switch crawl.Result.Status {
	case "running":
		return dto.NewFailedResponse[any](http.StatusConflict, "running crawl jobs cannot be deleted; cancel the crawl first", nil)
	case "completed", "errored", "cancelled_due_to_timeout", "cancelled_due_to_limits", "cancelled_by_user":
	default:
		return dto.NewFailedResponse[any](http.StatusBadGateway, types.ExceptionMessageBadGateway,
			fmt.Errorf("unexpected Cloudflare crawl status %q for job %s", crawl.Result.Status, website.JobId))
	}
	if err := dependencies.UnitOfWork.AIAgentWebsiteRepository().DeleteById(ctx, delete.Id); err != nil {
		return dto.NewFailedResponse[any](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
	}
	return dto.NewEmptyResponse(true, http.StatusOK)
}

func (Delete) APISettings() feature.APISettings {
	return feature.NewAPISettings(
		"Delete AI agent website",
		"Deletes a website in the business account only after Cloudflare confirms its crawl has completed, failed, or been cancelled. Running crawl jobs cannot be deleted.",
		types.HttpRequestTypeUri,
		http.MethodDelete,
		"/v1/ai-agent/websites/:id",
		true,
		true,
		types.APITagAIAgent,
		[]feature.APIError{
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageForbidden, http.StatusForbidden)),
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageUnauthorized, http.StatusUnauthorized)),
			feature.NewAPIError(*exception.NewCustomException("website not found", http.StatusNotFound)),
			feature.NewAPIError(*exception.NewCustomException("website has no crawl job or its crawl is still running", http.StatusConflict)),
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageBadGateway, http.StatusBadGateway)),
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageInternalServerError, http.StatusInternalServerError)),
		},
	)
}
