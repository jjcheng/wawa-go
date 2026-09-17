package repository

import (
	"context"
	"time"

	dao_wa "github.com/jjcheng/wawa-go/internal/dao/wa"
)

type WAMessageRepository interface {
	Repository[dao_wa.Message]
	List(ctx context.Context, phoneNumberId int32, customerId int32, ignoreUnsupportedType bool, page int, pageSize int) (messages []dao_wa.Message, totalPages int, totalCount int, err error)
	GetByWAMessageId(ctx context.Context, waMessageId string) (*dao_wa.Message, error)
	GetByToken(ctx context.Context, token string) (*dao_wa.Message, error)
	ListNeedResend(ctx context.Context) ([]dao_wa.Message, error)
	UpdateNextAttemptAt(ctx context.Context, id int32, claimUntil time.Time) (bool, error)
}
