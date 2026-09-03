package gormdb

import (
	dao_wa "github.com/jjcheng/wawa-go/internal/dao/wa"
	"github.com/jjcheng/wawa-go/internal/repository"
	"github.com/jjcheng/wawa-go/internal/service"
	"github.com/jjcheng/wawa-go/internal/types"
	"gorm.io/gorm"
)

type WASampleTemplateRepository struct {
	db     *gorm.DB
	logger *service.Logger
	repository.Repository[dao_wa.SampleTemplate]
}

func NewWASampleTemplateRepository(db *gorm.DB, logger *service.Logger) repository.WASampleTemplateRepository {
	return &WASampleTemplateRepository{
		db:         db,
		logger:     logger,
		Repository: NewRepository[dao_wa.SampleTemplate](db, logger),
	}
}

func (sampleTemplateRepository *WASampleTemplateRepository) List(category types.WATemplateCategory, language string) ([]dao_wa.SampleTemplate, error) {
	var templates []dao_wa.SampleTemplate
	query := sampleTemplateRepository.db.Model(&dao_wa.SampleTemplate{})
	if category != "" {
		query = query.Where("category = ?", category)
	}
	if language != "" {
		query = query.Where("language = ?", language)
	}
	if err := query.Order("id").Find(&templates).Error; err != nil {
		if sampleTemplateRepository.logger != nil {
			sampleTemplateRepository.logger.ErrorFunction(err, "wa.sample_templates", category, language)
		}
		return nil, err
	}
	return templates, nil
}
