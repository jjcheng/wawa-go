package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	dao_account "github.com/jjcheng/wawa-go/internal/dao/account"
	"github.com/jjcheng/wawa-go/internal/repository"
	"gorm.io/gorm"
)

type Cache struct {
	unitOfWork repository.UnitOfWork
}

func NewCache(unitOfWork repository.UnitOfWork) *Cache {
	return &Cache{
		unitOfWork: unitOfWork,
	}
}

func (cacheService *Cache) Get(ctx context.Context, key string) (string, error) {
	cache, err := cacheService.unitOfWork.AccountCacheRepository().GetByKey(ctx, key)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", CacheNotFoundError
		}
		return "", fmt.Errorf("Cache.Get key=%s error=%w", key, err)
	}
	return cache.Value, nil
}

func (cacheService *Cache) Set(ctx context.Context, key string, value string, expiresAt time.Time) error {
	cache := dao_account.Cache{
		Key:       key,
		Value:     value,
		ExpiresAt: expiresAt,
	}
	existing, err := cacheService.unitOfWork.AccountCacheRepository().GetByKey(ctx, key)
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("Cache.Set index=0 key=%s value=%s expiresAt=%v error=%w", key, value, expiresAt, err)
		}
	}
	if existing == nil {
		err = cacheService.unitOfWork.AccountCacheRepository().Insert(ctx, &cache)
	} else {
		err = cacheService.unitOfWork.AccountCacheRepository().UpdateFields(ctx, existing.Id, map[string]any{"value": value})
	}
	if err != nil {
		return fmt.Errorf("Cache.Set index=1 key=%s value=%s expiresAt=%v error=%w", key, value, expiresAt, err)
	}
	return nil
}

func (cacheService *Cache) Remove(ctx context.Context, key string) error {
	err := cacheService.unitOfWork.AccountCacheRepository().DeleteByKey(ctx, key)
	if err != nil {
		return fmt.Errorf("Cache.Remove key=%s error=%w", key, err)
	}
	return nil
}

// is scheduled at cmd/dispatcher/main.go
func (cacheService *Cache) CleanUp(ctx context.Context) error {
	err := cacheService.unitOfWork.AccountCacheRepository().DeleteExpired(ctx)
	if err != nil {
		return fmt.Errorf("Cache.CleanUp error=%w", err)
	}
	return nil
}

var CacheNotFoundError = errors.New("cache key not found")
