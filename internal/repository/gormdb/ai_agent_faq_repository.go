package gormdb

import (
	"context"
	"fmt"

	dao_ai_agent "github.com/jjcheng/wawa-go/internal/dao/ai_agent"
	"github.com/jjcheng/wawa-go/internal/repository"
	"github.com/jjcheng/wawa-go/internal/service"
	"gorm.io/gorm"
)

type AIAgentFAQRepository struct {
	db     *gorm.DB
	logger *service.Logger
	repository.Repository[dao_ai_agent.FAQ]
}

func NewAIAgentFAQRepository(db *gorm.DB, logger *service.Logger) repository.AIAgentFAQRepository {
	return &AIAgentFAQRepository{
		db:         db,
		logger:     logger,
		Repository: NewRepository[dao_ai_agent.FAQ](db, logger),
	}
}

func (faqRepository *AIAgentFAQRepository) ListByProfileId(ctx context.Context, profileId int32) ([]dao_ai_agent.FAQ, error) {
	var faqs []dao_ai_agent.FAQ
	if err := faqRepository.db.WithContext(ctx).
		Where("profile_id = ?", profileId).
		Order("id").
		Find(&faqs).Error; err != nil {
		return nil, fmt.Errorf("AIAgentFAQRepository.ListByProfileId profileId=%d error=%w", profileId, err)
	}
	return faqs, nil
}
