package gormdb

import (
	"context"
	"errors"

	dao_customer "github.com/jjcheng/wawa-go/internal/dao/customer"
	"github.com/jjcheng/wawa-go/internal/repository"
	"github.com/jjcheng/wawa-go/internal/service"
	"github.com/jjcheng/wawa-go/internal/types"
	"github.com/lib/pq"

	"gorm.io/gorm"
)

type CustomerRepository struct {
	db     *gorm.DB
	logger *service.Logger
	repository.Repository[dao_customer.Customer]
}

func NewCustomerRepository(db *gorm.DB, logger *service.Logger) repository.CustomerRepository {
	return &CustomerRepository{
		db:         db,
		logger:     logger,
		Repository: NewRepository[dao_customer.Customer](db, logger),
	}
}

func (customerRepository *CustomerRepository) CountActiveByUserId(ctx context.Context, userId int32) (int, error) {
	var count int64
	if err := customerRepository.db.WithContext(ctx).
		Model(&dao_customer.Customer{}).
		Where("user_id = ? AND status = ?", userId, types.CustomerStatusActive).
		Count(&count).Error; err != nil {
		customerRepository.logger.ErrorFunction(err, userId)
		return 0, err
	}
	return int(count), nil
}

func (customerRepository *CustomerRepository) CountActiveByMetaBusinessAccountId(ctx context.Context, metaBusinessAccountId string) (int, error) {
	var count int64
	if err := customerRepository.db.WithContext(ctx).
		Table("customer.customers").
		Joins("JOIN wa.user_phone_numbers ON wa.user_phone_numbers.user_id = customer.customers.user_id").
		Joins("JOIN wa.phone_numbers ON wa.phone_numbers.id = wa.user_phone_numbers.phone_number_id").
		Where("wa.phone_numbers.meta_waba_id = ? AND customer.customers.status = ?", metaBusinessAccountId, types.CustomerStatusActive).
		Distinct("customer.customers.id").
		Count(&count).Error; err != nil {
		customerRepository.logger.ErrorFunction(err, metaBusinessAccountId)
		return 0, err
	}
	return int(count), nil
}

func (customerRepository *CustomerRepository) GetByCountryCodePhoneNumber(ctx context.Context, userId int32, countryCode string, phoneNumber string) (*dao_customer.Customer, error) {
	var customer *dao_customer.Customer
	result := customerRepository.db.WithContext(ctx).Model(&dao_customer.Customer{}).Where("user_id = ? AND country_code = ? AND phone_number = ?", userId, countryCode, phoneNumber).First(&customer)
	if result.Error != nil {
		if !errors.Is(result.Error, gorm.ErrRecordNotFound) {
			customerRepository.logger.ErrorFunction(result.Error, userId, countryCode, phoneNumber)
		}
		return nil, result.Error
	}
	return customer, nil
}

func (customerRepository *CustomerRepository) GetByWAId(ctx context.Context, userId int32, waId string) (*dao_customer.Customer, error) {
	var customer dao_customer.Customer
	result := customerRepository.db.WithContext(ctx).
		Model(&dao_customer.Customer{}).
		Where("user_id = ? AND wa_id = ?", userId, waId).
		First(&customer)
	if result.Error != nil {
		if !errors.Is(result.Error, gorm.ErrRecordNotFound) {
			customerRepository.logger.ErrorFunction(result.Error, userId, waId)
		}
		return nil, result.Error
	}
	return &customer, nil
}

func (customerRepository *CustomerRepository) GetByMetaUserId(ctx context.Context, userId int32, bsuid string) (*dao_customer.Customer, error) {
	var customer *dao_customer.Customer
	result := customerRepository.db.WithContext(ctx).Model(&dao_customer.Customer{}).Where("user_id = ? AND meta_user_id = ?", userId, bsuid).First(&customer)
	if result.Error != nil {
		if !errors.Is(result.Error, gorm.ErrRecordNotFound) {
			customerRepository.logger.ErrorFunction(result.Error, userId, bsuid)
		}
		return nil, result.Error
	}
	return customer, nil
}

func (customerRepository *CustomerRepository) GetByWAIdOrMetaUserId(ctx context.Context, userId int32, waId string, metaUserId string) (*dao_customer.Customer, error) {
	var customer dao_customer.Customer
	query := customerRepository.db.WithContext(ctx).Model(&dao_customer.Customer{}).Where("user_id = ?", userId)
	if waId != "" && metaUserId != "" {
		query = query.Where("wa_id = ? OR meta_user_id = ?", waId, metaUserId)
	} else if waId != "" {
		query = query.Where("wa_id = ?", waId)
	} else {
		query = query.Where("meta_user_id = ?", metaUserId)
	}
	if err := query.First(&customer).Error; err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			customerRepository.logger.ErrorFunction(err, userId, waId, metaUserId)
		}
		return nil, err
	}
	return &customer, nil
}

func (customerRepository *CustomerRepository) ListByIds(ctx context.Context, userId int32, ids []int32) ([]dao_customer.Customer, error) {
	if len(ids) == 0 {
		return []dao_customer.Customer{}, nil
	}
	var customers []dao_customer.Customer
	result := customerRepository.db.WithContext(ctx).
		Model(&dao_customer.Customer{}).
		Where("user_id = ? AND id IN ?", userId, ids).
		Order("id").
		Find(&customers)
	if result.Error != nil {
		customerRepository.logger.ErrorFunction(result.Error, userId, ids)
		return nil, result.Error
	}
	return customers, nil
}

func (customerRepository *CustomerRepository) CountByIds(ctx context.Context, userId int32, ids []int32) (int, error) {
	if len(ids) == 0 {
		return 0, nil
	}
	var count int64
	result := customerRepository.db.WithContext(ctx).
		Model(&dao_customer.Customer{}).
		Where("user_id = ? AND id IN ?", userId, ids).
		Count(&count)
	if result.Error != nil {
		customerRepository.logger.ErrorFunction(result.Error, userId, ids)
		return 0, result.Error
	}
	return int(count), nil
}

func (customerRepository *CustomerRepository) GetByImportedPhoneNumber(ctx context.Context, userId int32, importedPhoneNumber string) (*dao_customer.Customer, error) {
	var customer dao_customer.Customer
	result := customerRepository.db.WithContext(ctx).
		Model(&dao_customer.Customer{}).
		Where("user_id = ? AND imported_phone_number = ?", userId, importedPhoneNumber).
		First(&customer)
	if result.Error != nil {
		if !errors.Is(result.Error, gorm.ErrRecordNotFound) {
			customerRepository.logger.ErrorFunction(result.Error, userId, importedPhoneNumber)
		}
		return nil, result.Error
	}
	return &customer, nil
}

func (customerRepository *CustomerRepository) GetDistinctTags(ctx context.Context, userId int32) ([]string, error) {
	var tags []string
	result := customerRepository.db.WithContext(ctx).
		Raw("SELECT DISTINCT unnest(tags) FROM customer.customers WHERE user_id = ? ORDER BY 1", userId).
		Scan(&tags)
	if result.Error != nil {
		customerRepository.logger.ErrorFunction(result.Error, userId)
		return nil, result.Error
	}
	return tags, nil
}

func (customerRepository *CustomerRepository) List(ctx context.Context, userId int32, name string, phoneNumber string, order types.OrderCustomersType, status types.CustomerStatus, tags []string, page int, pageSize int) (customers []dao_customer.Customer, totalItems int, totalPages int, err error) {
	query := customerRepository.db.WithContext(ctx).
		Model(&dao_customer.Customer{}).
		Where("user_id = ?", userId)
	if name != "" {
		query = query.Where("display_name ILIKE ?", "%"+name+"%")
	}
	if phoneNumber != "" {
		query = query.Where("phone_number ILIKE ?", "%"+phoneNumber+"%")
	}
	if status != "" {
		query = query.Where("status = ?", status)
	}
	if len(tags) > 0 {
		query = query.Where("tags && ?", pq.Array(tags))
	}
	var count int64
	if err := query.Count(&count).Error; err != nil {
		customerRepository.logger.ErrorFunction(err, userId, name, phoneNumber, order, status, tags)
		return nil, 0, 0, err
	}
	totalItems = int(count)
	totalPages = (totalItems + pageSize - 1) / pageSize
	if order == types.OrderCustomersTypeFromOld {
		query = query.Order("id")
	} else {
		query = query.Order("id DESC")
	}
	result := query.Offset((page - 1) * pageSize).Limit(pageSize).Find(&customers)
	if result.Error != nil {
		customerRepository.logger.ErrorFunction(result.Error, userId, name, phoneNumber, order, status, tags, page, pageSize)
		return nil, 0, 0, result.Error
	}
	return customers, totalItems, totalPages, nil
}
