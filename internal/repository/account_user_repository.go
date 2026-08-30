package repository

import (
	"context"

	dao_account "github.com/jjcheng/wawa-go/internal/dao/account"
)

type AccountUserRepository interface {
	Repository[dao_account.User]
	GetByEmailOrPhoneNumber(ctx context.Context, email string, phoneNumber string) (*dao_account.User, error)
	GetByPhoneNumber(ctx context.Context, phoneNumber string) (*dao_account.User, error)
	GetByAccessTokenHash(ctx context.Context, accessTokenHash string) (*dao_account.User, error)
	Get(ctx context.Context, id int32) (*dao_account.User, error)
}
