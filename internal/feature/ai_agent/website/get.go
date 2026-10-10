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

type Get struct {
	Id     int32 `uri:"id" val:"required" description:"id of the website"`
	Cursor int   `form:"cursor" description:"record pagination cursor, defaults to 0"`
	Limit  int   `form:"limit" description:"number of crawl records per page, defaults to 100"`
}

func (get *Get) Validate() []exception.InputException {
	var inputErrors []exception.InputException
	if get.Id <= 0 {
		inputErrors = append(inputErrors, exception.NewInputException("id", "invalid website id"))
	}
	if get.Cursor > 0 && get.Limit <= 0 {
		get.Limit = 30
	}
	return inputErrors
}

func (get Get) Handle(ctx context.Context, user *dto_account.User, dependencies *service.Dependencies) dto.Response[*service.CloudflareGetCrawlStatusResult] {
	if user == nil {
		return dto.NewFailedResponse[*service.CloudflareGetCrawlStatusResult](http.StatusForbidden, types.ExceptionMessageForbidden, nil)
	}
	if user.Type != types.UserTypeMaster {
		return dto.NewFailedResponse[*service.CloudflareGetCrawlStatusResult](http.StatusUnauthorized, types.ExceptionMessageUnauthorized, nil)
	}
	if inputErrors := get.Validate(); len(inputErrors) > 0 {
		return dto.NewInvalidInputResponse[*service.CloudflareGetCrawlStatusResult](inputErrors)
	}
	website, err := dependencies.UnitOfWork.AIAgentWebsiteRepository().GetById(ctx, get.Id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return dto.NewFailedResponse[*service.CloudflareGetCrawlStatusResult](http.StatusNotFound, "website not found", nil)
		}
		return dto.NewFailedResponse[*service.CloudflareGetCrawlStatusResult](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
	}
	profile, err := dependencies.UnitOfWork.AIAgentProfileRepository().GetById(ctx, website.ProfileId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return dto.NewFailedResponse[*service.CloudflareGetCrawlStatusResult](http.StatusNotFound, "profile not found", nil)
		}
		return dto.NewFailedResponse[*service.CloudflareGetCrawlStatusResult](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
	}
	if profile.BusinessAccountId != user.BusinessAccountId {
		return dto.NewFailedResponse[*service.CloudflareGetCrawlStatusResult](http.StatusUnauthorized, types.ExceptionMessageUnauthorized, nil)
	}
	if strings.TrimSpace(website.JobId) == "" {
		return dto.NewFailedResponse[*service.CloudflareGetCrawlStatusResult](http.StatusConflict, "website has no crawl job", fmt.Errorf("website %d has no crawl job ID", get.Id))
	}
	crawl, err := dependencies.Cloudflare.GetCrawlStatus(ctx, website.JobId, get.Cursor, get.Limit)
	if err != nil {
		return dto.NewFailedResponse[*service.CloudflareGetCrawlStatusResult](http.StatusBadGateway, types.ExceptionMessageBadGateway, err)
	}
	if !crawl.Success || strings.TrimSpace(crawl.Result.Status) == "" {
		return dto.NewFailedResponse[*service.CloudflareGetCrawlStatusResult](http.StatusBadGateway, types.ExceptionMessageBadGateway, fmt.Errorf("Cloudflare did not return a successful crawl status for job %s", website.JobId))
	}
	return dto.NewSuccessResponse(&crawl.Result)
}

func (Get) APISettings() feature.APISettings {
	return feature.NewAPISettings(
		"Get AI agent website",
		"Gets a website in the business account with its latest Cloudflare crawl status and a page of records. Use the returned crawl cursor to request the next page.",
		types.HttpRequestTypeUriQuery,
		http.MethodGet,
		"/v1/ai-agent/websites/:id",
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
