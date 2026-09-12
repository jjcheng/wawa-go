package repository

import (
	"context"

	dao_account "github.com/jjcheng/wawa-go/internal/dao/account"
)

type AccountUserRepository interface {
	Repository[dao_account.User]
	GetByPhoneNumber(ctx context.Context, countryCode string, phoneNumber string) (*dao_account.User, error)
	Get(ctx context.Context, id int32) (*dao_account.User, error)
	HasMasterUser(ctx context.Context, businessAccountId int32) (bool, error)
}
