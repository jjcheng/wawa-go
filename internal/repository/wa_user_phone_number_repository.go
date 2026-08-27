package repository

import (
	"context"

	dao_wa "github.com/jjcheng/wawa-go/internal/dao/wa"
)

type WAUserPhoneNumberRepository interface {
	Repository[dao_wa.UserPhoneNumber]
	ListPhoneNumbersByUserId(ctx context.Context, userId int32) ([]dao_wa.PhoneNumber, error)
	GetBusinessPortfolioAndAccountByUserId(ctx context.Context, userId int32) (*dao_wa.BusinessPortfolio, *dao_wa.BusinessAccount, error)
}
