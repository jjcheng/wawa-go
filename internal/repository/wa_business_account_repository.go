package repository

import (
	"context"

	dao_wa "github.com/jjcheng/wawa-go/internal/dao/wa"
)

type WABusinessAccountRepository interface {
	Repository[dao_wa.BusinessAccount]
	GetByMetaWABAId(ctx context.Context, metaWABAId string) (*dao_wa.BusinessAccount, *dao_wa.BusinessPortfolio, error)
}
