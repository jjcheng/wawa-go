package repository

import (
	"context"

	dao_account "github.com/jjcheng/wawa-go/internal/dao/account"
	"github.com/jjcheng/wawa-go/internal/types"
)

type AccountUserRepository interface {
	Repository[dao_account.User]
	GetByPhoneNumber(ctx context.Context, countryCode string, phoneNumber string) (*dao_account.User, error)
	HasMasterUserInBusinessAccount(ctx context.Context, businessAccountId int32) (bool, error)
	ListByBusinessAccountId(ctx context.Context, businessAccountId int32, typ *types.UserType) ([]dao_account.User, error)
	CountByBusinessAccountId(ctx context.Context, businessAccountId int32) (int, error)
	GetByIdsAndBusinessAccountId(ctx context.Context, ids []int32, businessAccountId int32) ([]dao_account.User, error)
}
