package gormdb

import (
	"context"
	"errors"

	dao_wa "github.com/jjcheng/wawa-go/internal/dao/wa"
	"github.com/jjcheng/wawa-go/internal/repository"
	"github.com/jjcheng/wawa-go/internal/service"
	"github.com/jjcheng/wawa-go/internal/types"
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

func (phoneNumberRepository *WAPhoneNumberRepository) Get(ctx context.Context, id int32) (*dao_wa.PhoneNumber, error) {
	var phoneNumber *dao_wa.PhoneNumber
	result := phoneNumberRepository.db.WithContext(ctx).Model(&dao_wa.PhoneNumber{}).Where("id = ?", id).First(&phoneNumber)
	if result.Error != nil {
		if !errors.Is(result.Error, gorm.ErrRecordNotFound) {
			phoneNumberRepository.logger.ErrorFunction(result.Error, id)
		}
		return nil, result.Error
	}
	if err := phoneNumberRepository.decryptSecrets(phoneNumber); err != nil {
		return nil, err
	}
	return phoneNumber, nil
}

func (phoneNumberRepository *WAPhoneNumberRepository) Insert(ctx context.Context, phoneNumber *dao_wa.PhoneNumber) error {
	displayPhoneNumber := phoneNumber.DisplayPhoneNumber
	waId := phoneNumber.WAId
	registrationPin := phoneNumber.RegistrationPin
	if err := phoneNumberRepository.encryptSecrets(phoneNumber); err != nil {
		return err
	}
	defer func() {
		phoneNumber.DisplayPhoneNumber = displayPhoneNumber
		phoneNumber.WAId = waId
		phoneNumber.RegistrationPin = registrationPin
	}()
	return phoneNumberRepository.Repository.Insert(ctx, phoneNumber)
}

func (phoneNumberRepository *WAPhoneNumberRepository) Update(ctx context.Context, phoneNumber *dao_wa.PhoneNumber) error {
	displayPhoneNumber := phoneNumber.DisplayPhoneNumber
	waId := phoneNumber.WAId
	registrationPin := phoneNumber.RegistrationPin
	if err := phoneNumberRepository.encryptSecrets(phoneNumber); err != nil {
		return err
	}
	defer func() {
		phoneNumber.DisplayPhoneNumber = displayPhoneNumber
		phoneNumber.WAId = waId
		phoneNumber.RegistrationPin = registrationPin
	}()
	return phoneNumberRepository.Repository.Update(ctx, phoneNumber)
}

func (phoneNumberRepository *WAPhoneNumberRepository) GetByWAIds(ctx context.Context, ids []int32) ([]dao_wa.PhoneNumber, error) {
	if len(ids) == 0 {
		return []dao_wa.PhoneNumber{}, nil
	}
	var phoneNumbers []dao_wa.PhoneNumber
	result := phoneNumberRepository.db.WithContext(ctx).
		Model(&dao_wa.PhoneNumber{}).
		Where("id IN ?", ids).
		Order("id").
		Find(&phoneNumbers)
	if result.Error != nil {
		phoneNumberRepository.logger.ErrorFunction(result.Error, ids)
		return nil, result.Error
	}
	for i := range phoneNumbers {
		if err := phoneNumberRepository.decryptSecrets(&phoneNumbers[i]); err != nil {
			return nil, err
		}
	}
	return phoneNumbers, nil
}

func (phoneNumberRepository *WAPhoneNumberRepository) GetByMetaPhoneNumberId(ctx context.Context, metaPhoneNumberId string) (*dao_wa.PhoneNumber, error) {
	var phoneNumber *dao_wa.PhoneNumber
	result := phoneNumberRepository.db.WithContext(ctx).Model(&dao_wa.PhoneNumber{}).Where("meta_phone_number_id = ?", metaPhoneNumberId).First(&phoneNumber)
	if result.Error != nil {
		if !errors.Is(result.Error, gorm.ErrRecordNotFound) {
			phoneNumberRepository.logger.ErrorFunction(result.Error, metaPhoneNumberId)
		}
		return nil, result.Error
	}
	if err := phoneNumberRepository.decryptSecrets(phoneNumber); err != nil {
		return nil, err
	}
	return phoneNumber, nil
}

func (phoneNumberRepository *WAPhoneNumberRepository) CountByBusinessAccountId(ctx context.Context, businessAccountId int32) (int, error) {
	var count int64
	if err := phoneNumberRepository.db.WithContext(ctx).
		Model(&dao_wa.PhoneNumber{}).
		Where("business_account_id = ?", businessAccountId).
		Count(&count).Error; err != nil {
		phoneNumberRepository.logger.ErrorFunction(err, businessAccountId)
		return 0, err
	}
	return int(count), nil
}

func (phoneNumberRepository *WAPhoneNumberRepository) ListByBusinessAccountId(ctx context.Context, businessAccountId int32, status types.WAPhoneNumberStatus, page int, pageSize int) (phoneNumbers []dao_wa.PhoneNumber, totalCount int, totalPages int, err error) {
	query := phoneNumberRepository.db.WithContext(ctx).
		Table("wa.phone_numbers AS pn").
		Joins("INNER JOIN account.users AS u ON u.id = pn.user_id").
		Where("pn.business_account_id = ?", businessAccountId)
	if status != "" {
		query = query.Where("pn.status = ?", status)
	}
	var count int64
	if err := query.Distinct("pn.id").Count(&count).Error; err != nil {
		phoneNumberRepository.logger.ErrorFunction(err, businessAccountId)
		return nil, 0, 0, err
	}
	totalCount = int(count)
	totalPages = (totalCount + pageSize - 1) / pageSize
	if err := query.Select("pn.*, u.name AS user_name").Order("pn.id").Offset((page - 1) * pageSize).Limit(pageSize).Find(&phoneNumbers).Error; err != nil {
		phoneNumberRepository.logger.ErrorFunction(err, businessAccountId)
		return nil, 0, 0, err
	}
	for i := range phoneNumbers {
		if err := phoneNumberRepository.decryptSecrets(&phoneNumbers[i]); err != nil {
			return nil, 0, 0, err
		}
	}
	return phoneNumbers, totalCount, totalPages, nil
}

func (phoneNumberRepository *WAPhoneNumberRepository) GetByUserId(ctx context.Context, userId int32) (*dao_wa.PhoneNumber, *dao_wa.BusinessAccount, *dao_wa.BusinessPortfolio, error) {
	var phoneNumber dao_wa.PhoneNumber
	result := phoneNumberRepository.db.WithContext(ctx).
		Model(&dao_wa.PhoneNumber{}).
		Where("user_id = ?", userId).
		First(&phoneNumber)
	if result.Error != nil {
		if !errors.Is(result.Error, gorm.ErrRecordNotFound) {
			phoneNumberRepository.logger.ErrorFunction(result.Error, userId)
		}
		return nil, nil, nil, result.Error
	}
	if err := phoneNumberRepository.decryptSecrets(&phoneNumber); err != nil {
		return nil, nil, nil, err
	}
	businessPortfolio, businessAccount, err := phoneNumberRepository.GetBusinessPortfolioAndAccountByUserId(ctx, userId)
	if err != nil {
		return nil, nil, nil, err
	}
	return &phoneNumber, businessAccount, businessPortfolio, nil
}

func (phoneNumberRepository *WAPhoneNumberRepository) decryptSecrets(phoneNumber *dao_wa.PhoneNumber) error {
	if phoneNumber.DisplayPhoneNumberEncrypted != "" {
		displayPhoneNumber, err := decryptStoredSecret(phoneNumber.DisplayPhoneNumberEncrypted, "wa.phone_numbers:display_phone_number:"+phoneNumber.MetaPhoneNumberId)
		if err != nil {
			phoneNumberRepository.logger.ErrorFunction(err, "phone_number", phoneNumber.Id)
			return err
		}
		phoneNumber.DisplayPhoneNumber = displayPhoneNumber
	}
	if phoneNumber.WAIdEncrypted != "" {
		waId, err := decryptStoredSecret(phoneNumber.WAIdEncrypted, "wa.phone_numbers:wa_id:"+phoneNumber.MetaPhoneNumberId)
		if err != nil {
			phoneNumberRepository.logger.ErrorFunction(err, "phone_number", phoneNumber.Id)
			return err
		}
		phoneNumber.WAId = waId
	}
	if phoneNumber.RegistrationPinEncrypted != "" {
		registrationPin, err := decryptStoredSecret(phoneNumber.RegistrationPinEncrypted, "wa.phone_numbers:registration_pin:"+phoneNumber.MetaPhoneNumberId)
		if err != nil {
			phoneNumberRepository.logger.ErrorFunction(err, "phone_number", phoneNumber.Id)
			return err
		}
		phoneNumber.RegistrationPin = registrationPin
	}
	return nil
}

func (phoneNumberRepository *WAPhoneNumberRepository) encryptSecrets(phoneNumber *dao_wa.PhoneNumber) error {
	if phoneNumber.DisplayPhoneNumber != "" {
		encrypted, err := encryptStoredSecret(phoneNumber.DisplayPhoneNumber, "wa.phone_numbers:display_phone_number:"+phoneNumber.MetaPhoneNumberId)
		if err != nil {
			return err
		}
		phoneNumber.DisplayPhoneNumberEncrypted = encrypted
		phoneNumber.DisplayPhoneNumber = ""
	}
	if phoneNumber.WAId != "" {
		hashed, err := hashStoredSecret(phoneNumber.WAId)
		if err != nil {
			return err
		}
		phoneNumber.WAIdHash = hashed
		encrypted, err := encryptStoredSecret(phoneNumber.WAId, "wa.phone_numbers:wa_id:"+phoneNumber.MetaPhoneNumberId)
		if err != nil {
			return err
		}
		phoneNumber.WAIdEncrypted = encrypted
		phoneNumber.WAId = ""
	}
	if phoneNumber.RegistrationPin != "" {
		encrypted, err := encryptStoredSecret(phoneNumber.RegistrationPin, "wa.phone_numbers:registration_pin:"+phoneNumber.MetaPhoneNumberId)
		if err != nil {
			return err
		}
		phoneNumber.RegistrationPinEncrypted = encrypted
		phoneNumber.RegistrationPin = ""
	}
	return nil
}

// used in update user
func (phoneNumberRepository *WAPhoneNumberRepository) GetBusinessPortfolioAndAccountByUserId(ctx context.Context, userId int32) (*dao_wa.BusinessPortfolio, *dao_wa.BusinessAccount, error) {
	var businessAccount dao_wa.BusinessAccount
	result := phoneNumberRepository.db.WithContext(ctx).
		Table("wa.business_accounts").
		Select("wa.business_accounts.*").
		Joins("JOIN wa.phone_numbers ON wa.phone_numbers.business_account_id = wa.business_accounts.id").
		Where("wa.phone_numbers.user_id = ?", userId).
		Order("wa.business_accounts.id").
		First(&businessAccount)
	if result.Error != nil {
		if !errors.Is(result.Error, gorm.ErrRecordNotFound) {
			phoneNumberRepository.logger.ErrorFunction(result.Error, userId)
		}
		return nil, nil, result.Error
	}
	var businessPortfolio dao_wa.BusinessPortfolio
	result = phoneNumberRepository.db.WithContext(ctx).
		Model(&dao_wa.BusinessPortfolio{}).
		Where("id = ?", businessAccount.BussinessPortfolioId).
		First(&businessPortfolio)
	if result.Error != nil {
		if !errors.Is(result.Error, gorm.ErrRecordNotFound) {
			phoneNumberRepository.logger.ErrorFunction(result.Error, userId, businessAccount.BussinessPortfolioId)
		}
		return nil, nil, result.Error
	}
	if err := decryptBusinessPortfolioAccessToken(&businessPortfolio); err != nil {
		return nil, nil, err
	}
	return &businessPortfolio, &businessAccount, nil
}

func decryptBusinessPortfolioAccessToken(businessPortfolio *dao_wa.BusinessPortfolio) error {
	if businessPortfolio.AccessTokenEncrypted == "" {
		return nil
	}
	accessToken, err := decryptStoredSecret(businessPortfolio.AccessTokenEncrypted, "wa.business_portfolios:access_token:"+businessPortfolio.MetaBusinessPortfolioId)
	if err != nil {
		return err
	}
	businessPortfolio.AccessToken = accessToken
	return nil
}
