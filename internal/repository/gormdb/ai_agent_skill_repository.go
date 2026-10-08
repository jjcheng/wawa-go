package gormdb

import (
	"context"
	"fmt"

	dao_ai_agent "github.com/jjcheng/wawa-go/internal/dao/ai_agent"
	"github.com/jjcheng/wawa-go/internal/repository"
	"github.com/jjcheng/wawa-go/internal/service"
	"gorm.io/gorm"
)

type AIAgentSkillRepository struct {
	db     *gorm.DB
	logger *service.Logger
	repository.Repository[dao_ai_agent.Skill]
}

func NewAIAgentSkillRepository(db *gorm.DB, logger *service.Logger) repository.AIAgentSkillRepository {
	return &AIAgentSkillRepository{
		db:         db,
		logger:     logger,
		Repository: NewRepository[dao_ai_agent.Skill](db, logger),
	}
}

func (skillRepository *AIAgentSkillRepository) ListByProfileId(ctx context.Context, profileId int32) ([]dao_ai_agent.Skill, error) {
	var skills []dao_ai_agent.Skill
	if err := skillRepository.db.WithContext(ctx).
		Where("profile_id = ?", profileId).
		Order("id").
		Find(&skills).Error; err != nil {
		return nil, fmt.Errorf("AIAgentSkillRepository.ListByProfileId profileId=%d error=%w", profileId, err)
	}
	return skills, nil
}
