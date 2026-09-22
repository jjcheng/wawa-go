package gormdb

import (
	dao_site "github.com/jjcheng/wawa-go/internal/dao/site"
	"github.com/jjcheng/wawa-go/internal/repository"
	"github.com/jjcheng/wawa-go/internal/service"
	"gorm.io/gorm"
)

type SiteFeedbackRepository struct {
	db     *gorm.DB
	logger *service.Logger
	repository.Repository[dao_site.Feedback]
}

func NewSiteFeedbackRepository(db *gorm.DB, logger *service.Logger) repository.SiteFeedbackRepository {
	return SiteFeedbackRepository{
		db:         db,
		logger:     logger,
		Repository: NewRepository[dao_site.Feedback](db, logger),
	}
}
