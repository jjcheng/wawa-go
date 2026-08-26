package repository

import (
	"context"

	dao_wa "github.com/jjcheng/wawa-go/internal/dao/wa"
	"github.com/jjcheng/wawa-go/internal/exception"
)

type WABusinessPortfolioRepository interface {
	Repository[dao_wa.BusinessPortfolio]
	Get(ctx context.Context, id int32) (*dao_wa.BusinessPortfolio, *exception.Exception)
	GetByMetaBusinessPortfolioId(ctx context.Context, metaBusinessPortfolioId string) (*dao_wa.BusinessPortfolio, *exception.Exception)
	ListByMetaBusinessPortfolioIds(ctx context.Context, metaBusinessPortfolioIds []string) ([]dao_wa.BusinessPortfolio, *exception.Exception)
}
