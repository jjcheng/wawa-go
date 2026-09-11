package repository

import (
	"context"

	dao_wa "github.com/jjcheng/wawa-go/internal/dao/wa"
)

type WABusinessPortfolioRepository interface {
	Repository[dao_wa.BusinessPortfolio]
	GetByMetaBusinessPortfolioId(ctx context.Context, metaBusinessPortfolioId string) (*dao_wa.BusinessPortfolio, error)
}
