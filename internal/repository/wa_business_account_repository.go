package repository

import (
	"context"

	dao_wa "github.com/jjcheng/wawa-go/internal/dao/wa"
)

type WABusinessAccountRepository interface {
	Repository[dao_wa.BusinessAccount]
	GetByWABAId(ctx context.Context, wabaId string) (*dao_wa.BusinessAccount, *dao_wa.BusinessPortfolio, error)
	ListByBusinessPortfolioId(ctx context.Context, businessPortfolioId int32) ([]dao_wa.BusinessAccount, error)
}
