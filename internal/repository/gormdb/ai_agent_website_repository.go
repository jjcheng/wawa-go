package gormdb

import (
	"context"
	"fmt"

	dao_ai_agent "github.com/jjcheng/wawa-go/internal/dao/ai_agent"
	"github.com/jjcheng/wawa-go/internal/repository"
	"github.com/jjcheng/wawa-go/internal/service"
	"gorm.io/gorm"
)

type AIAgentWebsiteRepository struct {
	db     *gorm.DB
	logger *service.Logger
	repository.Repository[dao_ai_agent.Website]
}

func NewAIAgentWebsiteRepository(db *gorm.DB, logger *service.Logger) repository.AIAgentWebsiteRepository {
	return &AIAgentWebsiteRepository{
		db:         db,
		logger:     logger,
		Repository: NewRepository[dao_ai_agent.Website](db, logger),
	}
}

func (websiteRepository *AIAgentWebsiteRepository) ListByBusinessAccountId(ctx context.Context, businessAccountId int32) ([]dao_ai_agent.Website, error) {
	var websites []dao_ai_agent.Website
	if err := websiteRepository.db.WithContext(ctx).
		Where("business_account_id = ?", businessAccountId).
		Order("id").
		Find(&websites).Error; err != nil {
		return nil, fmt.Errorf("AIAgentWebsiteRepository.ListByBusinessAccountId businessAccountId=%d error=%w", businessAccountId, err)
	}
	return websites, nil
}
