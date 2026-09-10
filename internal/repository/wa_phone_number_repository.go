package repository

import (
	"context"

	dao_wa "github.com/jjcheng/wawa-go/internal/dao/wa"
)

type WAPhoneNumberRepository interface {
	Repository[dao_wa.PhoneNumber]
	Get(ctx context.Context, id int32) (*dao_wa.PhoneNumber, error)
	GetByBusinessPortfolioId(ctx context.Context, id int32, businessPortfolioId int32) (*dao_wa.PhoneNumber, error)
	CountByMetaBusinessPortfolioId(ctx context.Context, metaBusinessPortfolioId string) (int, error)
	CountByMetaBusinessAccountId(ctx context.Context, metaBusinessAccountId string) (int, error)
	ListByMetaBusinessPortfolioId(ctx context.Context, metaBusinessPortfolioId string) ([]dao_wa.PhoneNumber, error)
	CheckExists(ctx context.Context, metaPhoneNumberId string) (bool, error)
	GetBusinessPortfolioByMetaPhoneNumberId(ctx context.Context, metaPhoneNmberId string) (*dao_wa.BusinessPortfolio, error)
	ListBusinessAccounts(ctx context.Context, id int32) ([]dao_wa.BusinessAccount, error)
	GetByPhoneNumberId(ctx context.Context, phoneNumberId string) (*dao_wa.PhoneNumber, error)
	ListByMetaBusinessAccountId(ctx context.Context, metaBusinessAccountId string, page int, pageSize int) (phoneNumbers []dao_wa.PhoneNumber, totalCount int, totalPages int, err error)
}
