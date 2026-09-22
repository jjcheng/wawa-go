package repository

import (
	"context"

	dao_account "github.com/jjcheng/wawa-go/internal/dao/account"
)

type AccountCacheRepository interface {
	Repository[dao_account.Cache]
	GetByKey(ctx context.Context, key string) (*dao_account.Cache, error)
	DeleteByKey(ctx context.Context, key string) error
	DeleteExpired(ctx context.Context) error
}
