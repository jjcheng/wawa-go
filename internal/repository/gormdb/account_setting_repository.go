package gormdb

import (
	dao_account "github.com/jjcheng/wawa-go/internal/dao/account"
	"github.com/jjcheng/wawa-go/internal/repository"
	"github.com/jjcheng/wawa-go/internal/service"

	"gorm.io/gorm"
)

type AISettingRepository struct {
	db     *gorm.DB
	logger *service.Logger
	repository.Repository[dao_account.Setting]
}

func NewAccountSettingRepository(db *gorm.DB, logger *service.Logger) repository.AccountSettingRepository {
	return &AISettingRepository{
		db:         db,
		logger:     logger,
		Repository: NewRepository[dao_account.Setting](db, logger),
	}
}
