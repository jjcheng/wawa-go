package repository

import (
	"context"

	dao_wa "github.com/jjcheng/wawa-go/internal/dao/wa"
	"github.com/jjcheng/wawa-go/internal/exception"
)

type WAPhoneNumberRepository interface {
	Repository[dao_wa.PhoneNumber]
	Get(ctx context.Context, id int32) (*dao_wa.PhoneNumber, *exception.Exception)
	GetByBusinessPortfolioId(ctx context.Context, id int32, businessPortfolioId int32) (*dao_wa.PhoneNumber, *exception.Exception)
	ListByBusinessPortfolioId(ctx context.Context, businessPortfolioId int32) (*[]dao_wa.PhoneNumber, *exception.Exception)
	CheckExists(ctx context.Context, metaPhoneNumberId string) (bool, *exception.Exception)
	GetBusinessPortfolioByMetaPhoneNumberId(ctx context.Context, metaPhoneNmberId string) (*dao_wa.BusinessPortfolio, *exception.Exception)
}
