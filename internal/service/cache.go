package service

import (
	"context"
	"errors"
	"fmt"
	"io"

	cloudflare "github.com/cloudflare/cloudflare-go/v4"
	"github.com/cloudflare/cloudflare-go/v4/kv"
	"github.com/cloudflare/cloudflare-go/v4/option"
	"github.com/jjcheng/wawa-go/internal/cfg"
	"github.com/jjcheng/wawa-go/internal/repository"
)

type Cache struct {
	client      *cloudflare.Client
	accountID   string
	namespaceID string
}

func NewCache(unitOfWork repository.UnitOfWork) *Cache {
	cacheService := &Cache{
		accountID:   cfg.Default().Cloudflare.AccountID,
		namespaceID: cfg.Default().Cloudflare.KVNamespaceID,
	}
	cacheService.client = cloudflare.NewClient(option.WithAPIToken(cfg.Default().Cloudflare.KVReadWriteAPIKey))
	return cacheService
}

func (cacheService *Cache) Get(ctx context.Context, key string) (string, error) {
	resp, err := cacheService.client.KV.Namespaces.Values.Get(ctx, cacheService.namespaceID, key, kv.NamespaceValueGetParams{AccountID: cloudflare.F(cacheService.accountID)})
	if err != nil {
		if isCloudflareNotFoundError(err) {
			return "", CacheNotFoundError
		}
		return "", fmt.Errorf("Cache.Get key=%s error=%w", key, err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("Cache.Get key=%s read error=%w", key, err)
	}
	return string(body), nil
}

func (cacheService *Cache) Set(ctx context.Context, key string, value string, expiresInSeconds int) error {
	params := kv.NamespaceValueUpdateParams{
		AccountID: cloudflare.F(cacheService.accountID),
		Value:     cloudflare.String(value),
	}
	if expiresInSeconds > 0 {
		params.ExpirationTTL = cloudflare.Float(float64(expiresInSeconds))
	} else {
		params.ExpirationTTL = cloudflare.Float(float64(300))
	}
	_, err := cacheService.client.KV.Namespaces.Values.Update(ctx, cacheService.namespaceID, key, params)
	if err != nil {
		return fmt.Errorf("Cache.Set key=%s value=%s expiresInSeconds=%v error=%w", key, value, expiresInSeconds, err)
	}
	return nil
}

// when key expires, will be deleted automatically
func (cacheService *Cache) Remove(ctx context.Context, key string) error {
	_, err := cacheService.client.KV.Namespaces.Values.Delete(ctx, cacheService.namespaceID, key, kv.NamespaceValueDeleteParams{AccountID: cloudflare.F(cacheService.accountID)})
	if err != nil && !isCloudflareNotFoundError(err) {
		return fmt.Errorf("Cache.Remove key=%s error=%w", key, err)
	}
	return nil
}

func isCloudflareNotFoundError(err error) bool {
	var apiErr *cloudflare.Error
	if !errors.As(err, &apiErr) {
		return false
	}
	return apiErr.StatusCode == 404
}

var CacheNotFoundError = errors.New("cache key not found")
