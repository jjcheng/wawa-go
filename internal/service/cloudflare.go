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
)

type Cloudflare struct {
	logger             *Logger
	turnstileSecretKey string
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
	}
	if cloudflare.turnstileSecretKey == "" {
		panic("missing secret key when creating cloudflare service")
	}
	if cloudflare.accountId == "" {
		panic("missing account id when creating cloudflare service")
	}
	if cloudflare.apiToken == "" {
		panic("missing api token when creating cloudflare service")
	}
	// cloudflare.client = cloudflare_sdk.NewClient(
	// 	option.WithAPIToken(cfg.Default().Cloudflare.APIToken),
	// )
	return nil
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
