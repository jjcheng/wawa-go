package repository

import (
	"context"

	dao_wa "github.com/jjcheng/wawa-go/internal/dao/wa"
)

type WAPhoneNumberRepository interface {
	Repository[dao_wa.PhoneNumber]
	Get(ctx context.Context, id int32) (*dao_wa.PhoneNumber, error)
	GetByBusinessPortfolioId(ctx context.Context, id int32, businessPortfolioId int32) (*dao_wa.PhoneNumber, error)
	ListByBusinessPortfolioId(ctx context.Context, businessPortfolioId int32) (*[]dao_wa.PhoneNumber, error)
	CheckExists(ctx context.Context, metaPhoneNumberId string) (bool, error)
	GetBusinessPortfolioByMetaPhoneNumberId(ctx context.Context, metaPhoneNmberId string) (*dao_wa.BusinessPortfolio, error)
	ListBusinessAccounts(ctx context.Context, id int32) ([]dao_wa.BusinessAccount, error)
}
