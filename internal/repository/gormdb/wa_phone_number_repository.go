package gormdb

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	dao_wa "github.com/jjcheng/wawa-go/internal/dao/wa"
	"github.com/jjcheng/wawa-go/internal/exception"
	"github.com/jjcheng/wawa-go/internal/repository"
	"github.com/jjcheng/wawa-go/internal/service"
	"gorm.io/gorm"
)

type WAPhoneNumberRepository struct {
	db     *gorm.DB
	logger *service.Logger
	repository.Repository[dao_wa.PhoneNumber]
}

func NewWAPhoneNumberRepository(db *gorm.DB, logger *service.Logger) repository.WAPhoneNumberRepository {
	return &WAPhoneNumberRepository{
		db:         db,
		logger:     logger,
		Repository: NewRepository[dao_wa.PhoneNumber](db, logger),
	}
}

func (phoneNumberRepository *WAPhoneNumberRepository) Get(ctx context.Context, id int32) (*dao_wa.PhoneNumber, *exception.Exception) {
	var phoneNumber *dao_wa.PhoneNumber
	result := phoneNumberRepository.db.WithContext(ctx).Model(&dao_wa.PhoneNumber{}).Where("id = ?", id).First(&phoneNumber)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, exception.NewCustomException("phone number not found", http.StatusNotFound)
		}
		phoneNumberRepository.logger.ErrorFunction(result.Error, id)
		return nil, exception.NewCustomException(fmt.Sprintf("error getting phone number: %v", result.Error), http.StatusInternalServerError)
	}
	return phoneNumber, nil
}

func (phoneNumberRepository *WAPhoneNumberRepository) GetByBusinessPortfolioId(ctx context.Context, id int32, businessPortfolioId int32) (*dao_wa.PhoneNumber, *exception.Exception) {
	var phoneNumber *dao_wa.PhoneNumber
	result := phoneNumberRepository.db.WithContext(ctx).Model(&dao_wa.PhoneNumber{}).Where("id = ? AND business_portfolio_id = ?", id, businessPortfolioId).First(&phoneNumber)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, exception.NewCustomException("phone number not found", http.StatusNotFound)
		}
		phoneNumberRepository.logger.ErrorFunction(result.Error, id, businessPortfolioId)
		return nil, exception.NewCustomException(fmt.Sprintf("error getting phone number: %v", result.Error), http.StatusInternalServerError)
	}
	return phoneNumber, nil
}

func (phoneNumberRepository *WAPhoneNumberRepository) ListByBusinessPortfolioId(ctx context.Context, businessPortfolioId int32) (*[]dao_wa.PhoneNumber, *exception.Exception) {
	var phoneNumbers []dao_wa.PhoneNumber
	result := phoneNumberRepository.db.WithContext(ctx).Model(&dao_wa.PhoneNumber{}).Where("business_portfolio_id = ?", businessPortfolioId).Order("id").Find(&phoneNumbers)
	if result.Error != nil {
		phoneNumberRepository.logger.ErrorFunction(result.Error, businessPortfolioId)
		return nil, exception.NewCustomException("error listing phone numbers", http.StatusInternalServerError)
	}
	return &phoneNumbers, nil
}

func (phoneNumberRepository *WAPhoneNumberRepository) CheckExists(ctx context.Context, metaPhoneNumberId string) (bool, *exception.Exception) {
	var count int64
	result := phoneNumberRepository.db.WithContext(ctx).Model(&dao_wa.PhoneNumber{}).Where("meta_phone_number_id = ?", metaPhoneNumberId).Count(&count)
	if result.Error != nil {
		phoneNumberRepository.logger.ErrorFunction(result.Error, metaPhoneNumberId)
		return false, exception.NewCustomException("error checking phone number exist", http.StatusInternalServerError)
	}
	return count > 0, nil
}
