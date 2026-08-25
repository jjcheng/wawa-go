package repository

import (
	"context"

	dao_wa "github.com/jjcheng/wawa-go/internal/dao/wa"
	"github.com/jjcheng/wawa-go/internal/exception"
)

type WABusinessAccountRepository interface {
	Repository[dao_wa.BusinessAccount]
	Get(ctx context.Context, id int32) (*dao_wa.BusinessAccount, *exception.Exception)
	GetByMetaWABAId(ctx context.Context, metaBusinessAccountId string) (*dao_wa.BusinessAccount, *exception.Exception)
	CheckExists(ctx context.Context, metaWABAId string) (bool, *exception.Exception)
}
