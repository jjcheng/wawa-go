package gormdb

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	dao_account "github.com/jjcheng/wawa-go/internal/dao/account"
	"github.com/jjcheng/wawa-go/internal/exception"
	"github.com/jjcheng/wawa-go/internal/repository"
	"github.com/jjcheng/wawa-go/internal/service"

	"gorm.io/gorm"
)

type AccountUserRepository struct {
	db     *gorm.DB
	logger *service.Logger
	repository.Repository[dao_account.User]
}

func NewAccountUserRepository(db *gorm.DB, logger *service.Logger) repository.AccountUserRepository {
	aiUserRepository := AccountUserRepository{
		db:         db,
		logger:     logger,
		Repository: NewRepository[dao_account.User](db, logger),
	}
	return &aiUserRepository
}

func (aiUserRepository *AccountUserRepository) Get(ctx context.Context, id int32) (*dao_account.User, *exception.Exception) {
	var item *dao_account.User
	result := aiUserRepository.db.Model(&dao_account.User{}).Where("id = ?", id).First(&item)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, exception.NewCustomException("user not found", http.StatusNotFound)
		}
		fmt.Printf("DATABASE ERROR in Get: %v\n", result.Error)
		aiUserRepository.logger.ErrorFunction(result.Error, id)
		return nil, exception.NewCustomException(fmt.Sprintf("error getting user: %v", result.Error), http.StatusInternalServerError)
	}
	return item, nil
}

func (aiUserRepository *AccountUserRepository) GetByPhoneNumber(ctx context.Context, phoneNumber string) (*dao_account.User, *exception.Exception) {
	var item *dao_account.User
	result := aiUserRepository.db.WithContext(ctx).Model(&dao_account.User{}).Where("phone_number = ?", phoneNumber).First(&item)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, exception.NewCustomException("user not found", http.StatusNotFound)
		}
		aiUserRepository.logger.ErrorFunction(result.Error, phoneNumber)
		return nil, exception.NewCustomException(fmt.Sprintf("error getting user: %v", result.Error), http.StatusInternalServerError)
	}
	return item, nil
}

func (aiUserRepository *AccountUserRepository) GetByEmailOrPhoneNumber(ctx context.Context, email string, phoneNumber string) (*dao_account.User, *exception.Exception) {
	var item *dao_account.User
	result := aiUserRepository.db.WithContext(ctx).Model(&dao_account.User{}).Where("email = ? OR phone_number = ?", email, phoneNumber).First(&item)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, exception.NewCustomException("user not found", http.StatusNotFound)
		}
		aiUserRepository.logger.ErrorFunction(result.Error, email, phoneNumber)
		return nil, exception.NewCustomException(fmt.Sprintf("error getting user: %v", result.Error), http.StatusInternalServerError)
	}
	return item, nil
}

func (aiUserRepository *AccountUserRepository) GetByOrganizationId(ctx context.Context, id int32, organizationId int32) (*dao_account.User, *exception.Exception) {
	var item *dao_account.User
	result := aiUserRepository.db.WithContext(ctx).Model(&dao_account.User{}).Where("id = ? AND organization_id = ?", id, organizationId).First(&item)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, exception.NewCustomException("user not found", http.StatusNotFound)
		}
		aiUserRepository.logger.ErrorFunction(result.Error, id, organizationId)
		return nil, exception.NewCustomException(fmt.Sprintf("error getting user: %v", result.Error), http.StatusInternalServerError)
	}
	return item, nil
}
