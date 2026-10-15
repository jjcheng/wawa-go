package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"net/url"
	"strings"

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
type CloudflareAISearchResponse struct {
	Success bool `json:"success"`
	Errors  []struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"errors"`
	Result json.RawMessage `json:"result"`
}

func (cloudflare *Cloudflare) CreateAISearchInstance(ctx context.Context, id string) (*CloudflareAISearchResponse, error) {
	apiUrl := fmt.Sprintf("%s/ai-search/namespaces/%s/instances", cloudflare.baseUrl, cloudflare.nameSpace)
	requestBody := map[string]any{
		"id":                    id,
		"hybrid_search_enabled": true,
		"embedding_model":       "@cf/qwen/qwen3-vl-embedding-2b",
		"reranking":             true,
		"reranking_model":       "@cf/baai/bge-reranker-base",
	}
	status, response, _, err := helper.RequestHTTP(ctx, apiUrl, "POST", cloudflare.getHeader(), &requestBody)
	if err != nil {
		return nil, fmt.Errorf("cloudflare create ai search instance error: %w", err)
	}
	return parseCloudflareAISearchResponse(status, response)
}

func (cloudflare *Cloudflare) UploadAISearchFile(ctx context.Context, instanceId string, file io.Reader, fileName string, contentType string) (*CloudflareAISearchResponse, error) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	header := make(textproto.MIMEHeader)
	header.Set("Content-Disposition", fmt.Sprintf(`form-data; name="file"; filename="%s"`, escapeMultipartQuotes(fileName)))
	if contentType != "" {
		header.Set("Content-Type", contentType)
	}
	part, err := writer.CreatePart(header)
	if err != nil {
		return nil, fmt.Errorf("cloudflare upload ai search file error: %w", err)
	}
	if _, err := io.Copy(part, file); err != nil {
		return nil, fmt.Errorf("cloudflare upload ai search file error: %w", err)
	}
	if err := writer.WriteField("wait_for_completion", "false"); err != nil {
		return nil, fmt.Errorf("cloudflare upload ai search file error: %w", err)
	}
	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("cloudflare upload ai search file error: %w", err)
	}
	apiUrl := fmt.Sprintf("%s/ai-search/namespaces/%s/instances/%s/items", cloudflare.baseUrl, cloudflare.nameSpace, url.PathEscape(instanceId))
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, apiUrl, body)
	if err != nil {
		return nil, fmt.Errorf("cloudflare upload ai search file error: %w", err)
	}
	request.Header.Set("Accept", "application/json")
	request.Header.Set("Content-Type", writer.FormDataContentType())
	for key, value := range *cloudflare.getHeader() {
		request.Header.Set(key, value)
	}
	httpResponse, err := http.DefaultClient.Do(request)
	if err != nil {
		return nil, fmt.Errorf("cloudflare upload ai search file error: %w", err)
	}
	defer httpResponse.Body.Close()
	responseBody, err := io.ReadAll(httpResponse.Body)
	if err != nil {
		return nil, fmt.Errorf("cloudflare upload ai search file error: %w", err)
	}
	text := string(responseBody)
	return parseCloudflareAISearchResponse(httpResponse.StatusCode, &text)
}

func parseCloudflareAISearchResponse(status int, response *string) (*CloudflareAISearchResponse, error) {
	if status < 200 || status > 299 {
		return nil, fmt.Errorf("cloudflare ai search response not 2xx: %d", status)
	}
	var result CloudflareAISearchResponse
	if response == nil {
		return nil, fmt.Errorf("cloudflare ai search returned an empty response")
	}
	if err := json.Unmarshal([]byte(*response), &result); err != nil {
		return nil, fmt.Errorf("error decoding cloudflare ai search response: %w", err)
	}
	if !result.Success {
		return nil, fmt.Errorf("cloudflare ai search failed: %+v", result.Errors)
	}
	return &result, nil
}

func escapeMultipartQuotes(value string) string {
	return strings.NewReplacer("\\", "\\\\", `"`, "\\\"").Replace(value)
}

// crawl
func (cloudflare *Cloudflare) Crawl(ctx context.Context, url string, includeSubdmomains bool, includePatterns []string, excludePatterns []string) (*CloudflareCrawlResponse, error) {
	apiUrl := fmt.Sprintf("%s/browser-run/crawl", cloudflare.baseUrl)
	requestBody := map[string]any{
		"url":           url,
		"crawlPurposes": []string{"search", "ai-input"},
		"formats":       []string{"markdown"},
		"source":        "all",
	}
	options := map[string]any{
		"includeSubdomains": includeSubdmomains,
	}
	if len(excludePatterns) > 0 {
		options["excludePatterns"] = excludePatterns
	}
	if len(includePatterns) > 0 {
		options["inxludePatterns"] = includePatterns
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
