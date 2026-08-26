package gormdb

import (
	"context"
	"time"

	dao_account "github.com/jjcheng/wawa-go/internal/dao/account"
	"github.com/jjcheng/wawa-go/internal/repository"
	"github.com/jjcheng/wawa-go/internal/service"

	"gorm.io/gorm"
)

type AccountSessionRepository struct {
	db *gorm.DB
	repository.Repository[dao_account.Session]
}

func NewAccountSessionRepository(db *gorm.DB, logger *service.Logger) repository.AccountSessionRepository {
	return &AccountSessionRepository{
		db:         db,
		Repository: NewRepository[dao_account.Session](db, logger),
	}
}

func (sessionRepository *AccountSessionRepository) GetUserByTokenHash(ctx context.Context, tokenHash string) (*dao_account.User, error, bool) {
	var user dao_account.User
	result := sessionRepository.db.WithContext(ctx).
		Table("account.sessions").
		Select("account.users.*").
		Joins("JOIN account.users ON account.users.id = account.sessions.user_id").
		Where("account.sessions.token_hash = ? AND account.sessions.revoked_at IS NULL AND account.sessions.expires_at > ? AND account.users.status <> ?", tokenHash, time.Now(), "INACTIVE").
		First(&user)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, nil, true
		}
		return nil, result.Error, false
	}
	return &user, nil, false
}

func (sessionRepository *AccountSessionRepository) RevokeByTokenHash(ctx context.Context, tokenHash string) error {
	now := time.Now()
	return sessionRepository.db.WithContext(ctx).Model(&dao_account.Session{}).
		Where("token_hash = ? AND revoked_at IS NULL", tokenHash).
		Updates(map[string]any{"revoked_at": now, "last_update": now}).Error
}
