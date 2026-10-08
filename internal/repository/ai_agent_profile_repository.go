package repository

import (
	"context"

	dao_ai_agent "github.com/jjcheng/wawa-go/internal/dao/ai_agent"
)

type AIProfileRepository interface {
	Repository[dao_ai_agent.Profile]
	ListByBusinessAccountId(ctx context.Context, businessAccountId int32) ([]dao_ai_agent.Profile, error)
	UpdateBusinessInfo(ctx context.Context, profile *dao_ai_agent.Profile) error
}
