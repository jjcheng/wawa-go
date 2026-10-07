package repository

import (
	"context"

	dao_ai_worker "github.com/jjcheng/wawa-go/internal/dao/ai_worker"
)

type AIWorkerMessageRepository interface {
	Repository[dao_ai_worker.Message]
	ListByConversationId(ctx context.Context, conversationId int32) ([]dao_ai_worker.Message, error)
}
