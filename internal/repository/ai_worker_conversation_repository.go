package repository

import (
	"context"

	dao_ai_worker "github.com/jjcheng/wawa-go/internal/dao/ai_worker"
)

type AIWorkerConversationRepository interface {
	Repository[dao_ai_worker.Conversation]
	ListByUserId(ctx context.Context, userId int32) ([]dao_ai_worker.Conversation, error)
}
