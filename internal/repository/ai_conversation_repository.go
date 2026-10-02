package repository

import (
	"context"

	dao_ai "github.com/jjcheng/wawa-go/internal/dao/ai"
)

type AIConversationRepository interface {
	Repository[dao_ai.Conversation]
	ListByUserId(ctx context.Context, userId int32) ([]dao_ai.Conversation, error)
}
