package repository

import (
	"context"

	dao_cm "github.com/jjcheng/wawa-go/internal/dao/cm"
	"github.com/jjcheng/wawa-go/internal/exception"
	"github.com/jjcheng/wawa-go/internal/types"
)

type CMMessageRepository interface {
	Repository[dao_cm.Message]
	GetByIdentifier(ctx context.Context, identifier string) (*dao_cm.Message, *exception.Exception)
	ListLastMessagesByConversationIds(ctx context.Context, conversationIds []int32, roles []types.ChatMessageRole) ([]dao_cm.Message, *exception.Exception)
	ListByUserIdAndConversationId(ctx context.Context, userId int32, conversationId int32, roles []types.ChatMessageRole) ([]dao_cm.Message, *exception.Exception)
	ListByUserIdAndConversationIdentifier(ctx context.Context, userId int32, conversationIdentifier string, roles []types.ChatMessageRole) ([]dao_cm.Message, *exception.Exception)
	GetByConversationIdAndLatestRole(ctx context.Context, conversationId int32, role types.ChatMessageRole) (*dao_cm.Message, *exception.Exception)
	GetByConversationIdentifierAndLatestRole(ctx context.Context, userId int32, conversationIdentifier string, role types.ChatMessageRole) (*dao_cm.Message, *exception.Exception)
	DeleteBySessionId(ctx context.Context, userId int32, sessionId string) *exception.Exception
}
