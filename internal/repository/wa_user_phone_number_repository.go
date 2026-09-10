package repository

import (
	"context"

	dao_wa "github.com/jjcheng/wawa-go/internal/dao/wa"
)

type WAUserPhoneNumberRepository interface {
	Repository[dao_wa.UserPhoneNumber]
	ListPhoneNumbersByUserId(ctx context.Context, userId int32, page int, pageSize int) (phoneNumbers []dao_wa.PhoneNumber, totalCount int, totalPages int, err error)
	ListPhoneNumbersByUserIdAndWAIds(ctx context.Context, userId int32, waIds []string) ([]dao_wa.PhoneNumber, error)
	CountPhoneNumbersByUserId(ctx context.Context, userId int32) (int, error)
	GetBusinessPortfolioAndAccountByUserId(ctx context.Context, userId int32) (*dao_wa.BusinessPortfolio, *dao_wa.BusinessAccount, error)
	GetValidBusinessAccount(ctx context.Context, userId int32, metaWABAId string) (*dao_wa.BusinessAccount, error)
	GetByUserIdPhoneNumberId(ctx context.Context, userId int32, phoneNumberId int32) (*dao_wa.UserPhoneNumber, error)
	GetByPhoneNumberId(ctx context.Context, phoneNumberId string) (*dao_wa.UserPhoneNumber, error)
}
