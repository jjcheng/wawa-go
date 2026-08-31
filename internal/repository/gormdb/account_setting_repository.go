package gormdb

import (
	"context"
	"errors"
	"net/http"

	dao_account "github.com/jjcheng/wawa-go/internal/dao/account"
	"github.com/jjcheng/wawa-go/internal/exception"
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

func (aiSettingRepository *AISettingRepository) ListByOrganizationId(ctx context.Context, organizationId int32) ([]dao_account.Setting, *exception.Exception) {
	var items []dao_account.Setting
	result := aiSettingRepository.db.Model(&dao_account.Setting{}).Where("organization_id = ?", organizationId).Order("id").Find(&items)
	if result.Error != nil {
		aiSettingRepository.logger.ErrorFunction(result.Error, organizationId)
		return nil, exception.NewCustomException("error listing settings", http.StatusInternalServerError)
	}
	return items, nil
}

func (aiSettingRepository *AISettingRepository) GetByOrganizationId(ctx context.Context, id int32, organizationId int32) (*dao_account.Setting, *exception.Exception) {
	var item *dao_account.Setting
	result := aiSettingRepository.db.Model(&dao_account.Setting{}).Where("id = ? AND organization_id = ?", id, organizationId).First(&item)
	if result.Error != nil {
		if !errors.Is(result.Error, gorm.ErrRecordNotFound) {
			aiSettingRepository.logger.ErrorFunction(result.Error, id, organizationId)
		}
		return nil, exception.NewCustomException("error getting setting", http.StatusInternalServerError)
	}
	return item, nil
}

func (aiSettingRepository *AISettingRepository) GetByNameAndOrganizationId(ctx context.Context, name string, organizationId int32) (*dao_account.Setting, *exception.Exception) {
	var item *dao_account.Setting
	result := aiSettingRepository.db.Model(&dao_account.Setting{}).Where("name = ? AND organization_id = ?", name, organizationId).First(&item)
	if result.Error != nil {
		if !errors.Is(result.Error, gorm.ErrRecordNotFound) {
			aiSettingRepository.logger.ErrorFunction(result.Error, name, organizationId)
		}
		return nil, exception.NewCustomException("error getting setting", http.StatusInternalServerError)
	}
	return item, nil
}
