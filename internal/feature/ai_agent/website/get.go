package feature_ai_agent_website

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/jjcheng/wawa-go/internal/dto"
	dto_account "github.com/jjcheng/wawa-go/internal/dto/account"
	dto_ai_agent "github.com/jjcheng/wawa-go/internal/dto/ai_agent"
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

type GetResult struct {
	Website dto_ai_agent.Website                   `json:"website"`
	Crawl   service.CloudflareGetCrawlStatusResult `json:"crawl"`
}

func (get *Get) Validate() []exception.InputException {
	var inputErrors []exception.InputException
	if get.Id <= 0 {
		inputErrors = append(inputErrors, exception.NewInputException("id", "invalid website id"))
	}
	if get.Cursor < 0 {
		inputErrors = append(inputErrors, exception.NewInputException("cursor", "cursor must not be negative"))
	}
	if get.Limit < 0 {
		inputErrors = append(inputErrors, exception.NewInputException("limit", "limit must not be negative"))
	} else if get.Limit == 0 {
		get.Limit = 100
	}
	return inputErrors
}

func (get Get) Handle(ctx context.Context, user *dto_account.User, dependencies *service.Dependencies) dto.Response[*GetResult] {
	if user == nil {
		return dto.NewFailedResponse[*GetResult](http.StatusForbidden, types.ExceptionMessageForbidden, nil)
	}
	if user.Type != types.UserTypeMaster {
		return dto.NewFailedResponse[*GetResult](http.StatusUnauthorized, types.ExceptionMessageUnauthorized, nil)
	}
	if inputErrors := get.Validate(); len(inputErrors) > 0 {
		return dto.NewInvalidInputResponse[*GetResult](inputErrors)
	}
	website, err := dependencies.UnitOfWork.AIAgentWebsiteRepository().GetById(ctx, get.Id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return dto.NewFailedResponse[*GetResult](http.StatusNotFound, "website not found", nil)
		}
		return dto.NewFailedResponse[*GetResult](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
	}
	if website.BusinessAccountId != user.BusinessAccountId {
		return dto.NewFailedResponse[*GetResult](http.StatusUnauthorized, types.ExceptionMessageUnauthorized, nil)
	}
	if strings.TrimSpace(website.JobId) == "" {
		return dto.NewFailedResponse[*GetResult](http.StatusConflict, "website has no crawl job", fmt.Errorf("website %d has no crawl job ID", get.Id))
	}
	crawl, err := dependencies.Cloudflare.GetCrawlStatus(ctx, website.JobId, get.Cursor, get.Limit)
	if err != nil {
		return dto.NewFailedResponse[*GetResult](http.StatusBadGateway, types.ExceptionMessageBadGateway, err)
	}
	if !crawl.Success || strings.TrimSpace(crawl.Result.Status) == "" {
		return dto.NewFailedResponse[*GetResult](http.StatusBadGateway, types.ExceptionMessageBadGateway, fmt.Errorf("Cloudflare did not return a successful crawl status for job %s", website.JobId))
	}
	result := &GetResult{
		Website: dto_ai_agent.NewWebsite(*website),
		Crawl:   crawl.Result,
	}
	result.Website.Status = crawl.Status()
	if result.Crawl.Records == nil {
		result.Crawl.Records = []service.CloudflareGetCrawlStatusRecord{}
	}
	return dto.NewSuccessResponse(result)
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
