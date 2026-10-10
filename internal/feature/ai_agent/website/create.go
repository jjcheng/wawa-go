package feature_ai_agent_website

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	dao_ai_agent "github.com/jjcheng/wawa-go/internal/dao/ai_agent"
	"github.com/jjcheng/wawa-go/internal/dto"
	dto_account "github.com/jjcheng/wawa-go/internal/dto/account"
	dto_ai_agent "github.com/jjcheng/wawa-go/internal/dto/ai_agent"
	"github.com/jjcheng/wawa-go/internal/exception"
	"github.com/jjcheng/wawa-go/internal/feature"
	"github.com/jjcheng/wawa-go/internal/service"
	"github.com/jjcheng/wawa-go/internal/types"
	"gorm.io/gorm"
)

type Create struct {
	ProfileId         int32    `uri:"id" val:"required" description:"id of the agent profile"`
	URL               string   `json:"url" val:"required" description:"HTTP or HTTPS URL of the website to crawl"`
	IncludeSubdomains bool     `json:"include_subdomains" description:"whether to crawl subdomains"`
	ExcludePatterns   []string `json:"exclude_patterns" description:"URL patterns to exclude from crawling" example:"[\"test\"]"`
	IncludePatterns   []string `json:"include_patterns" description:"put full urls to only include those pages"`
}

func (create *Create) Validate() []exception.InputException {
	var inputErrors []exception.InputException
	if create.ProfileId <= 0 {
		inputErrors = append(inputErrors, exception.NewInputException("profile_id", "invalid profile id"))
	}
	create.URL = strings.TrimSpace(create.URL)
	parsedURL, err := url.Parse(create.URL)
	if err != nil || parsedURL.Hostname() == "" || (parsedURL.Scheme != "http" && parsedURL.Scheme != "https") || parsedURL.User != nil {
		inputErrors = append(inputErrors, exception.NewInputException("url", "a valid HTTP or HTTPS URL without credentials is required"))
	}
	for i, pattern := range create.ExcludePatterns {
		create.ExcludePatterns[i] = strings.TrimSpace(pattern)
		if create.ExcludePatterns[i] == "" {
			inputErrors = append(inputErrors, exception.NewInputException("exclude_patterns", "exclude patterns must not be empty"))
			break
		}
	}
	return inputErrors
}

func (create Create) Handle(ctx context.Context, user *dto_account.User, dependencies *service.Dependencies) dto.Response[*dto_ai_agent.Website] {
	if user == nil {
		return dto.NewFailedResponse[*dto_ai_agent.Website](http.StatusForbidden, types.ExceptionMessageForbidden, nil)
	}
	if user.Type != types.UserTypeMaster {
		return dto.NewFailedResponse[*dto_ai_agent.Website](http.StatusUnauthorized, types.ExceptionMessageUnauthorized, nil)
	}
	if inputErrors := create.Validate(); len(inputErrors) > 0 {
		return dto.NewInvalidInputResponse[*dto_ai_agent.Website](inputErrors)
	}
	profile, err := dependencies.UnitOfWork.AIAgentProfileRepository().GetById(ctx, create.ProfileId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return dto.NewFailedResponse[*dto_ai_agent.Website](http.StatusNotFound, "profile not found", nil)
		}
		return dto.NewFailedResponse[*dto_ai_agent.Website](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
	}
	if profile.BusinessAccountId != user.BusinessAccountId {
		return dto.NewFailedResponse[*dto_ai_agent.Website](http.StatusUnauthorized, types.ExceptionMessageUnauthorized, nil)
	}
	crawl, err := dependencies.Cloudflare.Crawl(ctx, create.URL, create.IncludeSubdomains, create.IncludePatterns, create.ExcludePatterns)
	if err != nil {
		return dto.NewFailedResponse[*dto_ai_agent.Website](http.StatusBadGateway, types.ExceptionMessageBadGateway, err)
	}
	if !crawl.Success || strings.TrimSpace(crawl.Result) == "" {
		return dto.NewFailedResponse[*dto_ai_agent.Website](http.StatusBadGateway, types.ExceptionMessageBadGateway, fmt.Errorf("Cloudflare did not return a successful crawl job"))
	}
	website := dao_ai_agent.Website{
		ProfileId:         create.ProfileId,
		URL:               create.URL,
		IncludeSubdomains: create.IncludeSubdomains,
		IncludePatterns:   create.IncludePatterns,
		ExcludePatterns:   create.ExcludePatterns,
		JobId:             crawl.Result,
		Status:            "Running",
	}
	if err := dependencies.UnitOfWork.AIAgentWebsiteRepository().Insert(ctx, &website); err != nil {
		// Cancel the untracked job even if the request context has been cancelled.
		cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
		defer cancel()
		if cancelErr := dependencies.Cloudflare.CancelCrawl(cleanupCtx, crawl.Result); cancelErr != nil {
			err = errors.Join(err, fmt.Errorf("cancel untracked crawl job %s: %w", crawl.Result, cancelErr))
		}
		return dto.NewFailedResponse[*dto_ai_agent.Website](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
	}
	result := dto_ai_agent.NewWebsite(website)
	return dto.NewSuccessResponse(&result)
}

func (Create) APISettings() feature.APISettings {
	return feature.NewAPISettings(
		"Create AI agent website",
		"Starts a website crawl and saves the website in the AI agent profile's business account.",
		types.HttpRequestTypeUriJSON,
		http.MethodPost,
		"/v1/ai-agent/profiles/:id/websites",
		true,
		true,
		types.APITagAIAgent,
		[]feature.APIError{
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageForbidden, http.StatusForbidden)),
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageUnauthorized, http.StatusUnauthorized)),
			feature.NewAPIError(*exception.NewCustomException("profile not found", http.StatusNotFound)),
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageBadGateway, http.StatusBadGateway)),
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageInternalServerError, http.StatusInternalServerError)),
		},
	)
}
