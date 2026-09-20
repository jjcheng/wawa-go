package service

import (
	"context"
	"time"

	dao_account "github.com/jjcheng/wawa-go/internal/dao/account"
	"github.com/jjcheng/wawa-go/internal/repository"
)

type Cache struct {
	unitOfWork repository.UnitOfWork
}

func NewCache(unitOfWork repository.UnitOfWork) *Cache {
	return &Cache{
		unitOfWork: unitOfWork,
	}
}

func (cacheService *Cache) Insert(ctx context.Context, key string, value string, expiresAt time.Time) error {
	cache := dao_account.Cache{
		Key:       key,
		Value:     value,
		ExpiresAt: expiresAt,
	}
	return cacheService.unitOfWork.AccountCacheRepository().Insert(ctx, &cache)
}

func (cacheService *Cache) Get(ctx context.Context, key string) (string, error) {
	cache, err := cacheService.unitOfWork.AccountCacheRepository().GetByKey(ctx, key)
	if err != nil {
		return "", err
	}
	return cache, nil
}

func (cacheService *Cache) Remove(ctx context.Context, key string) error {
	return cacheService.unitOfWork.AccountCacheRepository().DeleteByKey(ctx, key)
}

// TODO: need to schedule clean up
func (cacheService *Cache) Cleanup(ctx context.Context) error {
	return cacheService.unitOfWork.AccountCacheRepository().DeleteExpired(ctx)
}
