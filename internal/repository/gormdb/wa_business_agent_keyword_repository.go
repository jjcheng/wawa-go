package gormdb

import (
	"context"
	"fmt"

	dao_wa "github.com/jjcheng/wawa-go/internal/dao/wa"
	"github.com/jjcheng/wawa-go/internal/repository"
	"github.com/jjcheng/wawa-go/internal/service"
	"gorm.io/gorm"
)

type WABusinessAgentKeywordRepository struct {
	db     *gorm.DB
	logger *service.Logger
	repository.Repository[dao_wa.BusinessAgentKeyword]
}

func NewWABusinessAgentKeywordRepository(db *gorm.DB, logger *service.Logger) repository.WABusinessAgentKeywordRespository {
	return WABusinessAgentKeywordRepository{
		db:         db,
		logger:     logger,
		Repository: NewRepository[dao_wa.BusinessAgentKeyword](db, logger),
	}
}

func (waBusinessAgentKeywordRepository WABusinessAgentKeywordRepository) ListByPhoneNumberId(ctx context.Context, phoneNumberId int32) ([]dao_wa.BusinessAgentKeyword, error) {
	var keywords []dao_wa.BusinessAgentKeyword
	if err := waBusinessAgentKeywordRepository.db.WithContext(ctx).
		Where("phone_number_id = ?", phoneNumberId).
		Order("id").
		Find(&keywords).Error; err != nil {
		return nil, fmt.Errorf("WABusinessAgentKeywordRepository.ListByPhoneNumberId phoneNumberId=%d error=%w", phoneNumberId, err)
	}
	return keywords, nil
}

func (waBusinessAgentKeywordRepository WABusinessAgentKeywordRepository) ListMatchingByPhoneNumberId(ctx context.Context, phoneNumberId int32, messageText string) ([]dao_wa.BusinessAgentKeyword, error) {
	var keywords []dao_wa.BusinessAgentKeyword
	if err := waBusinessAgentKeywordRepository.db.WithContext(ctx).
		Where("phone_number_id = ?", phoneNumberId).
		Where("char_length(keyword) <= char_length(?)", messageText).
		Where("strpos(lower(?), lower(keyword)) > 0", messageText).
		Order("id").
		Find(&keywords).Error; err != nil {
		return nil, fmt.Errorf("WABusinessAgentKeywordRepository.ListMatchingByPhoneNumberId phoneNumberId=%d error=%w", phoneNumberId, err)
	}
	return keywords, nil
}
