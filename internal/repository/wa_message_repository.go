package repository

import (
	"context"
	"time"

	dao_wa "github.com/jjcheng/wawa-go/internal/dao/wa"
)

type WAMessageRepository interface {
	Repository[dao_wa.Message]
	List(ctx context.Context, phoneNumberId string, customerMetaId string, customerPhoneNumber string, ignoreUnsupportedType bool, page int, pageSize int) (messages []dao_wa.Message, totalPages int, totalCount int, err error)
	CountOutgoingByUserIdSince(ctx context.Context, userId int32, since time.Time, deliveredOnly bool) (int, error)
	GetByWAMessageId(ctx context.Context, waMessageId string) (*dao_wa.Message, error)
}
