package repository

import (
	"context"

	dao_cm "github.com/jjcheng/wawa-go/internal/dao/cm"
	"github.com/jjcheng/wawa-go/internal/exception"
)

type CMCachedMessageRepository interface {
	Repository[dao_cm.CachedMessage]
	UpdateReplied(ctx context.Context, identifier string) *exception.Exception
	CheckHasUnreplied(ctx context.Context, conversationIdentifier string) bool
	UpdateAllUnreplied(ctx context.Context, conversatonIdentifier string) *exception.Exception
}
