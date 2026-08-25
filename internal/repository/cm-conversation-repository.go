package repository

import (
	"context"
	"time"

	dao_cm "github.com/jjcheng/wawa-go/internal/dao/cm"
	"github.com/jjcheng/wawa-go/internal/exception"
)

type CMConversationRepository interface {
	Repository[dao_cm.Conversation]
	GetByUserIdAndId(ctx context.Context, userId int32, id int32) (*dao_cm.Conversation, *exception.Exception)
	GetByUserIdAndIdentifier(ctx context.Context, userId int32, identifier string) (*dao_cm.Conversation, *exception.Exception)
	CheckConversationExist(ctx context.Context, userId int32, identifier string) (bool, *exception.Exception)
	ListByUserId(ctx context.Context, userId int32) ([]dao_cm.Conversation, *exception.Exception)
	GetStartContext(ctx context.Context, userId int32, identifier string, rateLimitFromDateTime time.Time) (*dao_cm.Conversation, int, *exception.Exception)
	UpdateTitle(ctx context.Context, id int32, title string) *exception.Exception
}
