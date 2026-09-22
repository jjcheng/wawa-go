package gormdb

import (
	"context"
	"errors"
	"fmt"
	"time"

	dao_account "github.com/jjcheng/wawa-go/internal/dao/account"
	"github.com/jjcheng/wawa-go/internal/repository"
	"github.com/jjcheng/wawa-go/internal/service"
	"gorm.io/gorm"
)

type AccountSessionRepository struct {
	db     *gorm.DB
	logger *service.Logger
	repository.Repository[dao_account.Session]
}

func NewAccountSessionRepository(db *gorm.DB, logger *service.Logger) repository.AccountSessionRepository {
	accountSessionRepository := AccountSessionRepository{
		db:         db,
		logger:     logger,
		Repository: NewRepository[dao_account.Session](db, logger),
	}
	return &accountSessionRepository
}

func (accountSessionRepository *AccountSessionRepository) GetByAccessTokenHash(ctx context.Context, accessTokenHash string) (*dao_account.Session, error) {
	var item dao_account.Session
	result := accountSessionRepository.db.WithContext(ctx).
		Model(&dao_account.Session{}).
		Where("access_token_hashed = ?", accessTokenHash).
		First(&item)
	if result.Error != nil {
		if !errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("AccountSessionRepository.GetByAccessTokenHash accessTokenHash=%s error=%w", accessTokenHash, result.Error)
		}
		return nil, result.Error
	}
	return &item, nil
}

func (accountSessionRepository *AccountSessionRepository) UpdateLastUsed(ctx context.Context, id int32) error {
	err := accountSessionRepository.UpdateFields(ctx, id, map[string]any{"last_used_at": time.Now()})
	if err != nil {
		return fmt.Errorf("AccountSessionRepository.UpdateLastUsed id=%d error=%w", id, err)
	}
	return nil
}

func (accountSessionRepository *AccountSessionRepository) UpdateRevokedAt(ctx context.Context, id int32) error {
	err := accountSessionRepository.UpdateFields(ctx, id, map[string]any{"revoked_at": time.Now()})
	if err != nil {
		return fmt.Errorf("AccountSessionRepository.UpdateRevokedAt id=%d error=%w", id, err)
	}
	return nil
}

func (accountSessionRepository *AccountSessionRepository) DeleteByUserId(ctx context.Context, userId int32) error {
	if err := accountSessionRepository.db.WithContext(ctx).
		Where("user_id = ?", userId).
		Delete(&dao_account.Session{}).Error; err != nil {
		return fmt.Errorf("AccountSessionRepository.DeleteByUserId userId=%d error=%w", userId, err)
	}
	return nil
}
