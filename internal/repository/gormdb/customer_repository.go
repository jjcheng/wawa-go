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

func (customerRepository *CustomerRepository) GetByBSUID(ctx context.Context, userId int32, bsuid string) (*dao_customer.Customer, error) {
	var customer *dao_customer.Customer
	result := customerRepository.db.WithContext(ctx).Model(&dao_customer.Customer{}).Where("user_id = ? AND bsuid = ?", userId, bsuid).First(&customer)
	if result.Error != nil {
		if !errors.Is(result.Error, gorm.ErrRecordNotFound) {
			customerRepository.logger.ErrorFunction(result.Error, userId, bsuid)
		}
		return nil, result.Error
	}
	return customer, nil
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

func (customerRepository *CustomerRepository) List(ctx context.Context, userId int32, order types.OrderCustomersType, tags []string, page int, pageSize int) ([]dao_customer.Customer, error) {
	var customers []dao_customer.Customer
	offset := (page - 1) * pageSize
	query := customerRepository.db.WithContext(ctx).
		Model(&dao_customer.Customer{}).
		Where("user_id = ?", userId)
	if len(tags) > 0 {
		query = query.Where("tags && ?", pq.Array(tags))
	}
	if order == types.OrderCustomersTypeFromOld {
		query = query.Order("id")
	} else {
		query = query.Order("id DESC")
	}
	result := query.Offset(offset).Limit(pageSize).Find(&customers)
	if result.Error != nil {
		customerRepository.logger.ErrorFunction(result.Error, userId, order, tags, page, pageSize)
		return nil, result.Error
	}
	return customers, nil
}
