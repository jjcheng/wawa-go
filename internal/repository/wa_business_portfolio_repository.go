package repository

import (
	"context"

	dao_wa "github.com/jjcheng/wawa-go/internal/dao/wa"
)

type WABusinessPortfolioRepository interface {
	Repository[dao_wa.BusinessPortfolio]
	Get(ctx context.Context, id int32) (*dao_wa.BusinessPortfolio, error)
	GetByUserId(ctx context.Context, userId int32) (*dao_wa.BusinessPortfolio, error)
	GetByMetaBusinessPortfolioId(ctx context.Context, metaBusinessPortfolioId string) (*dao_wa.BusinessPortfolio, error)
	ListByMetaBusinessPortfolioIds(ctx context.Context, metaBusinessPortfolioIds []string) ([]dao_wa.BusinessPortfolio, error)
}
