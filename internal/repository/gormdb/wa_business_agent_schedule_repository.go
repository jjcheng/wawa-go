package gormdb

import (
	dao_wa "github.com/jjcheng/wawa-go/internal/dao/wa"
	"github.com/jjcheng/wawa-go/internal/repository"
	"github.com/jjcheng/wawa-go/internal/service"
	"gorm.io/gorm"
)

type WABusinessAgentScheduleRepository struct {
	db     *gorm.DB
	logger *service.Logger
	repository.Repository[dao_wa.BusinessAgentSchedule]
}

func NewWABusinessAgentScheduleRepository(db *gorm.DB, logger *service.Logger) repository.WABusinessAgentScheduleRepository {
	return &WABusinessAgentScheduleRepository{
		db:         db,
		logger:     logger,
		Repository: NewRepository[dao_wa.BusinessAgentSchedule](db, logger),
	}
}
