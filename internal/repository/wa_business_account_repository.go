package repository

import (
	"context"

	dao_wa "github.com/jjcheng/wawa-go/internal/dao/wa"
)

type WABusinessAccountRepository interface {
	Repository[dao_wa.BusinessAccount]
	Get(ctx context.Context, id int32) (*dao_wa.BusinessAccount, error)
	GetByMetaWABAId(ctx context.Context, metaWABAId string) (*dao_wa.BusinessAccount, error)
	CheckExists(ctx context.Context, metaWABAId string) (bool, error)
	ListByMetaWABAIds(ctx context.Context, metaWABAIds []string) ([]dao_wa.BusinessAccount, error)
	GetBusinessPortfolioByWABAId(ctx context.Context, metaWABAId string) (*dao_wa.BusinessPortfolio, error)
}
