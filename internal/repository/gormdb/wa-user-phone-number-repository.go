package gormdb

import (
	dao_wa "github.com/jjcheng/wawa-go/internal/dao/wa"
	"github.com/jjcheng/wawa-go/internal/repository"
	"github.com/jjcheng/wawa-go/internal/service"
	"gorm.io/gorm"
)

type WAUserPhoneNumberRepository struct {
	db     *gorm.DB
	logger *service.Logger
	repository.Repository[dao_wa.UserPhoneNumber]
}

func NewWAUserPhoneNumberRepository(db *gorm.DB, logger *service.Logger) repository.WAUserPhoneNumberRepository {
	return &WAUserPhoneNumberRepository{
		db:         db,
		logger:     logger,
		Repository: NewRepository[dao_wa.UserPhoneNumber](db, logger),
	}
}
