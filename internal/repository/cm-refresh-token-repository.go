package repository

import (
	"context"

	dao_cm "github.com/jjcheng/wawa-go/internal/dao/cm"
	"github.com/jjcheng/wawa-go/internal/exception"
)

type CMRefreshTokenRepository interface {
	Repository[dao_cm.RefreshToken]
	GetByTokenHash(ctx context.Context, tokenHash string) (*dao_cm.RefreshToken, *exception.Exception)
	UpdateLastUsed(ctx context.Context, tokenId int32, ipAddress *string, userAgent *string) *exception.Exception
	RevokeToken(ctx context.Context, tokenId int32, reason string) *exception.Exception
	RevokeByTokenHash(ctx context.Context, tokenHash string, reason string) *exception.Exception
	RevokeAllForUser(ctx context.Context, userId int32, reason string) *exception.Exception
	DeleteExpired(ctx context.Context) (int64, *exception.Exception)
}
