package repository

import (
	"context"

	dao_ai_agent "github.com/jjcheng/wawa-go/internal/dao/ai_agent"
)

type AIAgentWebsiteRepository interface {
	Repository[dao_ai_agent.Website]
	ListByProfileId(ctx context.Context, profileId int32) ([]dao_ai_agent.Website, error)
}
