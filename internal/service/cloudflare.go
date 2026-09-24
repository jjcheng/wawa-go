package service

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/jjcheng/wawa-go/internal/cfg"
)

type Cloudflare struct {
	logger    *Logger
	secretKey string
}

func NewCloudflare(logger *Logger) *Cloudflare {
	cloudflare := &Cloudflare{
		logger:    logger,
		secretKey: cfg.Default().Cloudflare.TurnstileSecretKey,
	}
	if cloudflare.secretKey == "" {
		panic("missing secret key when creating cloudflare service")
	}
	return cloudflare
}

func (cloudflare *Cloudflare) VerifyTurnstile(token, remoteIp string) (bool, error) {
	const siteVerifyURL = "https://challenges.cloudflare.com/turnstile/v0/siteverify"
	form := url.Values{}
	form.Set("secret", cloudflare.secretKey)
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
