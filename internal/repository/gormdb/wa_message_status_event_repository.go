package gormdb

import (
	dao_wa "github.com/jjcheng/wawa-go/internal/dao/wa"
	"github.com/jjcheng/wawa-go/internal/repository"
	"github.com/jjcheng/wawa-go/internal/service"
	"gorm.io/gorm"
)

type WAMessageStatusEventRepository struct {
	db     *gorm.DB
	logger *service.Logger
	repository.Repository[dao_wa.MessageStatusEvent]
}

func NewWAMessageStatusEventRepository(db *gorm.DB, logger *service.Logger) repository.WAMessageStatusEventRepository {
	return &WAMessageStatusEventRepository{
		db:         db,
		logger:     logger,
		Repository: NewRepository[dao_wa.MessageStatusEvent](db, logger),
	}
}
