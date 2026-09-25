package repository

import (
	"context"

	dao_account "github.com/jjcheng/wawa-go/internal/dao/account"
)

type AccountUserPhoneNumberRepository interface {
	Repository[dao_account.UserPhoneNumber]
	ListByUserId(ctx context.Context, userId int32) ([]dao_account.UserPhoneNumber, error)
	ListByPhoneNumberId(ctx context.Context, phoneNumberId int32) ([]dao_account.UserPhoneNumber, error)
	ListByUserIds(ctx context.Context, userIds []int32) ([]dao_account.UserPhoneNumber, error)
	ListByPhoneNumberIds(ctx context.Context, phoneNumberId []int32) ([]dao_account.UserPhoneNumber, error)
	CountByBusinessAccountIdAndPhoneNumberIds(ctx context.Context, businessAccountId int32, phoneNumberIds []int32) (int, error)
	DeleteByUserId(ctx context.Context, userId int32) error
	DeleteByPhoneNumberId(ctx context.Context, phoneNumberId int32) error
	GetByUserIdAndPhoneNumberId(ctx context.Context, userId int32, phoneNumberId int32) (*dao_account.UserPhoneNumber, error)
}
