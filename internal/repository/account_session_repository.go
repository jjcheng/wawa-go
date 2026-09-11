package repository

import (
	"context"

	dao_account "github.com/jjcheng/wawa-go/internal/dao/account"
)

type AccountSessionRepository interface {
	Repository[dao_account.Session]
	GetByAccessTokenHash(ctx context.Context, accessTokenHash string) (*dao_account.Session, error)
	UpdateLastUsed(ctx context.Context, id int32) error
	UpdateRevokedAt(ctx context.Context, id int32) error
}
