package gormdb

import (
	"context"
	"time"

	dao_cm "github.com/jjcheng/wawa-go/internal/dao/cm"
	"github.com/jjcheng/wawa-go/internal/exception"
	"github.com/jjcheng/wawa-go/internal/repository"
	"github.com/jjcheng/wawa-go/internal/service"

	"gorm.io/gorm"
)

type CMRefreshTokenRepository struct {
	db     *gorm.DB
	logger *service.Logger
	repository.Repository[dao_cm.RefreshToken]
}

func NewCMRefreshTokenRepository(db *gorm.DB, logger *service.Logger) *CMRefreshTokenRepository {
	return &CMRefreshTokenRepository{
		db:         db,
		logger:     logger,
		Repository: NewRepository[dao_cm.RefreshToken](db, logger),
	}
}

func (r *CMRefreshTokenRepository) GetByTokenHash(ctx context.Context, tokenHash string) (*dao_cm.RefreshToken, *exception.Exception) {
	var token dao_cm.RefreshToken
	if err := r.db.WithContext(ctx).Where("token_hash = ? AND is_revoked = false", tokenHash).First(&token).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, exception.NewCustomException("refresh token not found", 404)
		}
		r.logger.ErrorFunction(err, tokenHash)
		return nil, exception.NewCustomException("error getting refresh token", 500)
	}
	return &token, nil
}

func (r *CMRefreshTokenRepository) GetByUserId(ctx context.Context, userId int32) ([]dao_cm.RefreshToken, *exception.Exception) {
	var tokens []dao_cm.RefreshToken
	if err := r.db.WithContext(ctx).Where("user_id = ?", userId).Find(&tokens).Error; err != nil {
		r.logger.ErrorFunction(err, userId)
		return nil, exception.NewCustomException("error getting refresh tokens", 500)
	}
	return tokens, nil
}

func (r *CMRefreshTokenRepository) GetByTokenFamily(ctx context.Context, tokenFamily string) ([]dao_cm.RefreshToken, *exception.Exception) {
	var tokens []dao_cm.RefreshToken
	if err := r.db.WithContext(ctx).Where("token_family = ?", tokenFamily).Find(&tokens).Error; err != nil {
		r.logger.ErrorFunction(err, tokenFamily)
		return nil, exception.NewCustomException("error getting refresh tokens by family", 500)
	}
	return tokens, nil
}

func (r *CMRefreshTokenRepository) UpdateLastUsed(ctx context.Context, tokenId int32, ipAddress *string, userAgent *string) *exception.Exception {
	now := time.Now()
	updates := map[string]any{
		"last_used_at": now,
		"ip_address":   ipAddress,
		"user_agent":   userAgent,
	}

	if err := r.db.WithContext(ctx).Model(&dao_cm.RefreshToken{}).Where("id = ?", tokenId).Updates(updates).Error; err != nil {
		r.logger.ErrorFunction(err, tokenId)
		return exception.NewCustomException("error updating refresh token", 500)
	}
	return nil
}

func (r *CMRefreshTokenRepository) RevokeToken(ctx context.Context, tokenId int32, reason string) *exception.Exception {
	now := time.Now()
	updates := map[string]any{
		"is_revoked":     true,
		"revoked_at":     now,
		"revoked_reason": reason,
	}

	if err := r.db.WithContext(ctx).Model(&dao_cm.RefreshToken{}).Where("id = ?", tokenId).Updates(updates).Error; err != nil {
		r.logger.ErrorFunction(err, tokenId)
		return exception.NewCustomException("error revoking refresh token", 500)
	}
	return nil
}

func (r *CMRefreshTokenRepository) RevokeByTokenHash(ctx context.Context, tokenHash string, reason string) *exception.Exception {
	now := time.Now()
	updates := map[string]any{
		"is_revoked":     true,
		"revoked_at":     now,
		"revoked_reason": reason,
	}

	result := r.db.WithContext(ctx).Model(&dao_cm.RefreshToken{}).Where("token_hash = ?", tokenHash).Updates(updates)
	if result.Error != nil {
		r.logger.ErrorFunction(result.Error, tokenHash)
		return exception.NewCustomException("error revoking refresh token by hash", 500)
	}
	return nil
}

func (r *CMRefreshTokenRepository) RevokeAllForUser(ctx context.Context, userId int32, reason string) *exception.Exception {
	now := time.Now()
	updates := map[string]any{
		"is_revoked":     true,
		"revoked_at":     now,
		"revoked_reason": reason,
	}

	if err := r.db.WithContext(ctx).Model(&dao_cm.RefreshToken{}).Where("user_id = ?", userId).Updates(updates).Error; err != nil {
		r.logger.ErrorFunction(err, userId)
		return exception.NewCustomException("error revoking refresh tokens for user", 500)
	}
	return nil
}

func (r *CMRefreshTokenRepository) DeleteExpired(ctx context.Context) (int64, *exception.Exception) {
	result := r.db.WithContext(ctx).Where("expires_at < ?", time.Now()).Delete(&dao_cm.RefreshToken{})
	if result.Error != nil {
		r.logger.ErrorFunction(result.Error)
		return 0, exception.NewCustomException("error deleting expired refresh tokens", 500)
	}
	return result.RowsAffected, nil
}
