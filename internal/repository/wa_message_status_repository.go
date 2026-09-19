package repository

import (
	"context"

	dao_wa "github.com/jjcheng/wawa-go/internal/dao/wa"
)

type WAMessageStatusRepository interface {
	Repository[dao_wa.MessageStatus]
	ListByMessageId(ctx context.Context, messageId int32) ([]dao_wa.MessageStatus, error)
}
