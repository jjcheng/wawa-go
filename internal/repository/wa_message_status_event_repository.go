package repository

import (
	"context"

	dao_wa "github.com/jjcheng/wawa-go/internal/dao/wa"
)

type WAMessageStatusEventRepository interface {
	Repository[dao_wa.MessageStatusEvent]
	ListByMessageId(ctx context.Context, messageId int32) ([]dao_wa.MessageStatusEvent, error)
}
