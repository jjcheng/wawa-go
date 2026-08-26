package repository

import (
	"context"

	dao_account "github.com/jjcheng/wawa-go/internal/dao/account"
)

type AccountSessionRepository interface {
	Repository[dao_account.Session]
	GetUserByTokenHash(ctx context.Context, tokenHash string) (*dao_account.User, error, bool)
	RevokeByTokenHash(ctx context.Context, tokenHash string) error
}
