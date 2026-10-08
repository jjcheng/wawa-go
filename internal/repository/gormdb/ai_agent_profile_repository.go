package gormdb

import (
	"context"
	"fmt"
	"time"

	dao_ai_agent "github.com/jjcheng/wawa-go/internal/dao/ai_agent"
	"github.com/jjcheng/wawa-go/internal/repository"
	"github.com/jjcheng/wawa-go/internal/service"
	"gorm.io/gorm"
)

type AIAgentProfileRepository struct {
	db     *gorm.DB
	logger *service.Logger
	repository.Repository[dao_ai_agent.Profile]
}

func (profileRepository *AIAgentProfileRepository) UpdateBusinessInfo(ctx context.Context, profile *dao_ai_agent.Profile) error {
	profile.LastUpdatedAt = time.Now()
	result := profileRepository.db.WithContext(ctx).
		Model(profile).
		Where("id = ? AND business_account_id = ?", profile.Id, profile.BusinessAccountId).
		Select("business_info", "last_updated_at").
		Updates(profile)
	if result.Error != nil {
		return fmt.Errorf("AIAgentProfileRepository.UpdateBusinessInfo profileId=%d error=%w", profile.Id, result.Error)
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func NewAIAgentProfileRepository(db *gorm.DB, logger *service.Logger) repository.AIProfileRepository {
	return &AIAgentProfileRepository{
		db:         db,
		logger:     logger,
		Repository: NewRepository[dao_ai_agent.Profile](db, logger),
	}
}

func (profileRepository *AIAgentProfileRepository) ListByBusinessAccountId(ctx context.Context, businessAccountId int32) ([]dao_ai_agent.Profile, error) {
	var profiles []dao_ai_agent.Profile
	if err := profileRepository.db.WithContext(ctx).
		Where("business_account_id = ?", businessAccountId).
		Order("id").
		Find(&profiles).Error; err != nil {
		return nil, fmt.Errorf("AIProfileRepository.ListByBusinessAccountId businessAccountId=%d error=%w", businessAccountId, err)
	}
	return profiles, nil
}
