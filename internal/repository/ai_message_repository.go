package repository

import (
	"context"

	dao_ai "github.com/jjcheng/wawa-go/internal/dao/ai"
)

type AIMessageRepository interface {
	Repository[dao_ai.Message]
	ListByConversationId(ctx context.Context, conversationId int32) ([]dao_ai.Message, error)
}
