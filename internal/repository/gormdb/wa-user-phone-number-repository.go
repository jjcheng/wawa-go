package gormdb

import (
	"context"
	"fmt"
	"net/http"

	dao_wa "github.com/jjcheng/wawa-go/internal/dao/wa"
	"github.com/jjcheng/wawa-go/internal/exception"
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

func (userPhoneNumberRepository *WAUserPhoneNumberRepository) ListPhoneNumbersByUserId(ctx context.Context, userId int32) ([]dao_wa.PhoneNumber, *exception.Exception) {
	var phoneNumbers []dao_wa.PhoneNumber
	result := userPhoneNumberRepository.db.WithContext(ctx).
		Table("wa.phone_numbers").
		Select("wa.phone_numbers.*").
		Joins("JOIN wa.user_phone_numbers ON wa.user_phone_numbers.phone_number_id = wa.phone_numbers.id").
		Where("wa.user_phone_numbers.user_id = ?", userId).
		Order("wa.phone_numbers.id").
		Find(&phoneNumbers)
	if result.Error != nil {
		userPhoneNumberRepository.logger.ErrorFunction(result.Error, userId)
		return nil, exception.NewCustomException(fmt.Sprintf("error listing user phone numbers: %v", result.Error), http.StatusInternalServerError)
	}
	return phoneNumbers, nil
}
