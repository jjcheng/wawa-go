package repository

import (
	"context"

	dao_wa "github.com/jjcheng/wawa-go/internal/dao/wa"
)

type WAMessageStatusEventRepository interface {
	Repository[dao_wa.MessageStatusEvent]
	List(ctx context.Context, waMessageId string) ([]dao_wa.MessageStatusEvent, error)
	GetLatest(ctx context.Context, waMessageId string) (*dao_wa.MessageStatusEvent, error)
}
