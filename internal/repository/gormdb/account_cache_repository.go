package gormdb

import (
	"context"
	"errors"
	"fmt"

	dao_account "github.com/jjcheng/wawa-go/internal/dao/account"
	"github.com/jjcheng/wawa-go/internal/repository"
	"github.com/jjcheng/wawa-go/internal/service"
	"gorm.io/gorm"
)

type AccountCacheRepository struct {
	db     *gorm.DB
	logger *service.Logger
	repository.Repository[dao_account.Cache]
}

func NewAccountCacheRepository(db *gorm.DB, logger *service.Logger) repository.AccountCacheRepository {
	accountCacheRepository := AccountCacheRepository{
		db:         db,
		logger:     logger,
		Repository: NewRepository[dao_account.Cache](db, logger),
	}
	return &accountCacheRepository
}

func (accountCacheRepository *AccountCacheRepository) GetByKey(ctx context.Context, key string) (*dao_account.Cache, error) {
	var cache dao_account.Cache
	result := accountCacheRepository.db.WithContext(ctx).Where("key = ?", key).First(&cache)
	if result.Error != nil {
		if !errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("AccountCacheRepository.GetByKey key=%s error=%w", key, result.Error)
		}
		return nil, result.Error
	}
	return &cache, nil
}

func (accountCacheRepository *AccountCacheRepository) DeleteByKey(ctx context.Context, key string) error {
	if err := accountCacheRepository.db.WithContext(ctx).Exec("delete from account.cache where key = ?", key).Error; err != nil {
		return fmt.Errorf("AccountCacheRepository.DeleteByKey key=%s error=%w", key, err)
	}
	return nil
}

func (accountCacheRepository *AccountCacheRepository) DeleteExpired(ctx context.Context) error {
	if err := accountCacheRepository.db.WithContext(ctx).Exec("delete from account.cache where expires_at <= now()").Error; err != nil {
		return fmt.Errorf("AccountCacheRepository.DeleteExpired error=%w", err)
	}
	return nil
}
