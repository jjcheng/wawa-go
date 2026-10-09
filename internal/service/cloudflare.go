package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	cloudflare_sdk "github.com/cloudflare/cloudflare-go/v7"
	"github.com/cloudflare/cloudflare-go/v7/ai_search"
	"github.com/jjcheng/wawa-go/internal/cfg"
	"github.com/jjcheng/wawa-go/internal/helper"
)

type Cloudflare struct {
	logger             *Logger
	turnstileSecretKey string
	baseUrl            string
	accountId          string
	apiToken           string
	nameSpace          string
	client             *cloudflare_sdk.Client
}

func NewCloudflare(logger *Logger) *Cloudflare {
	cloudflare := &Cloudflare{
		logger:             logger,
		turnstileSecretKey: cfg.Default().Cloudflare.TurnstileSecretKey,
		accountId:          cfg.Default().Cloudflare.AccountId,
		apiToken:           cfg.Default().Cloudflare.APIToken,
		nameSpace:          "default",
		baseUrl:            fmt.Sprintf("https://api.cloudflare.com/client/v4/accounts/%s", cfg.Default().Cloudflare.AccountId),
	}
	// cloudflare.client = cloudflare_sdk.NewClient(
	// 	option.WithAPIToken(cfg.Default().Cloudflare.APIToken),
	// )
	return cloudflare
}

// turnstile
func (cloudflare *Cloudflare) VerifyTurnstile(token, remoteIp string) (bool, error) {
	const siteVerifyURL = "https://challenges.cloudflare.com/turnstile/v0/siteverify"
	form := url.Values{}
	form.Set("secret", cloudflare.turnstileSecretKey)
	form.Set("response", token)
	if remoteIp != "" {
		form.Set("remoteip", remoteIp)
	}
	resp, err := http.PostForm(siteVerifyURL, form)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	var result struct {
		Success    bool     `json:"success"`
		ErrorCodes []string `json:"error-codes"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return false, fmt.Errorf("failed to decode turstile response body: %w", err)
	}
	if !result.Success {
		return false, fmt.Errorf("turnstile verification failed: %v", result.ErrorCodes)
	}
	return true, nil
}

// ai search
func (cloudflare *Cloudflare) CreateAISearchInstance(ctx context.Context, id string) (*ai_search.NamespaceInstanceNewResponse, error) {
	response, err := cloudflare.client.AISearch.Namespaces.Instances.New(ctx, cloudflare.nameSpace, ai_search.NamespaceInstanceNewParams{
		AccountID:           cloudflare_sdk.F(cloudflare.accountId),
		HybridSearchEnabled: cloudflare_sdk.F(true),
		ID:                  cloudflare_sdk.String(id),
		EmbeddingModel:      cloudflare_sdk.String("@cf/qwen/qwen3-vl-embedding-2b"),
		Reranking:           cloudflare_sdk.Bool(true),
		RerankingModel:      cloudflare_sdk.String("@cf/baai/bge-reranker-base"),
		//	Metadata:            cloudflare,
	})
	return response, err
}

func (cloudflare *Cloudflare) UploadAISearchFile(ctx context.Context, instanceId string, file io.Reader, fileName string, contentType string) (*ai_search.NamespaceInstanceItemUploadResponse, error) {
	response, err := cloudflare.client.AISearch.Namespaces.Instances.Items.Upload(ctx, cloudflare.nameSpace, instanceId, ai_search.NamespaceInstanceItemUploadParams{
		AccountID: cloudflare_sdk.String(cloudflare.accountId),
		File: ai_search.NamespaceInstanceItemUploadParamsFile{
			File:              cloudflare_sdk.FileParam(file, fileName, contentType),
			WaitForCompletion: cloudflare_sdk.Bool(false),
		},
	})
	return response, err
}

// crawl
func (cloudflare *Cloudflare) Crawl(ctx context.Context, url string, includeSubdmomains bool, excludePatterns []string) (*CloudflareCrawlResponse, error) {
	apiUrl := fmt.Sprintf("%s/browser-run/crawl", cloudflare.baseUrl)
	requestBody := map[string]any{
		"url":           url,
		"crawlPurposes": []string{"search"},
		"contentUse":    "reference",
		"formats":       []string{"markdown"},
		"source":        "all",
	}
	options := map[string]any{
		"includeSubdomains": includeSubdmomains,
	}
	if len(excludePatterns) > 0 {
		options["excludePatterns"] = excludePatterns
	}
	requestBody["options"] = options
	status, response, _, err := helper.RequestHTTP(ctx, apiUrl, "POST", cloudflare.getHeader(), &requestBody)
	if err != nil {
		return nil, fmt.Errorf("cloudflare crawl error: %w", err)
	}
	if status != 200 {
		return nil, fmt.Errorf("cloudflare crawl response not 200: %d", status)
	}
	var result CloudflareCrawlResponse
	if err := json.Unmarshal([]byte(*response), &result); err != nil {
		return nil, fmt.Errorf("error decoding clourflare crawl response: %w", err)
	}
	if !result.Success {

	}
	return &result, nil
}

func (cloudflare *Cloudflare) GetCrawlStatus(ctx context.Context, id string, cursor int, limit int) (*CloudflareGetCrawlStatusResponse, error) {
	apiUrl := fmt.Sprintf("%s/browser-run/crawl/%s?cursor=%d&limit=%d", cloudflare.baseUrl, id, cursor, limit)
	status, response, _, err := helper.RequestHTTP(ctx, apiUrl, "GET", cloudflare.getHeader(), nil)
	if err != nil {
		return nil, fmt.Errorf("error getting cloudflare crawl job status: %w", err)
	}
	if status != 200 {
		return nil, fmt.Errorf("get Cloudflare crawl job status response not 200: %d", status)
	}
	var result CloudflareGetCrawlStatusResponse
	if err := json.Unmarshal([]byte(*response), &result); err != nil {
		return nil, fmt.Errorf("error parsing cloudflare get crawl job status response: %w", err)
	}
	return &result, nil
}

func (cloudflare *Cloudflare) CancelCrawl(ctx context.Context, id string) error {
	apiUrl := fmt.Sprintf("%s/browser-run/crawl/%s", cloudflare.baseUrl, id)
	status, response, _, err := helper.RequestHTTP(ctx, apiUrl, "DELETE", cloudflare.getHeader(), nil)
	if err != nil {
		return fmt.Errorf("error cancel cloudflare crawl job status: %w", err)
	}
	if status != 200 {
		return fmt.Errorf("cancel cloudflare crawl job response not 200: %d", status)
	}
	var result struct {
		Success bool `json:"success"`
		Errors  []struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"errors"`
	}
	if err := json.Unmarshal([]byte(*response), &result); err != nil {
		return fmt.Errorf("decode Cloudflare crawl cancellation response: %w", err)
	}
	if !result.Success {
		return fmt.Errorf("Cloudflare crawl cancellation failed: %+v", result.Errors)
	}
	return nil
}

// shared
func (cloudflare *Cloudflare) getHeader() *map[string]string {
	return &map[string]string{
		"Authorization": fmt.Sprintf("Bearer %s", cloudflare.apiToken),
	}
}
