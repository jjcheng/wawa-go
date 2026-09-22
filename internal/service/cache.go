package service

import (
	"context"
	"errors"
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
		return "", err
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
			return err
		}
	}
	if existing == nil {
		return cacheService.unitOfWork.AccountCacheRepository().Insert(ctx, &cache)
	} else {
		return cacheService.unitOfWork.AccountCacheRepository().UpdateFields(ctx, existing.Id, map[string]any{"value": value})
	}
}

func (cacheService *Cache) Remove(ctx context.Context, key string) error {
	return cacheService.unitOfWork.AccountCacheRepository().DeleteByKey(ctx, key)
}

// is scheduled at cmd/dispatcher/main.go
func (cacheService *Cache) Cleanup(ctx context.Context) error {
	return cacheService.unitOfWork.AccountCacheRepository().DeleteExpired(ctx)
}

var CacheNotFoundError = errors.New("cache key not found")
