package repository

import (
	"context"

	dao_customer "github.com/jjcheng/wawa-go/internal/dao/customer"
	"github.com/jjcheng/wawa-go/internal/types"
)

type CustomerRepository interface {
	Repository[dao_customer.Customer]
	GetByCountryCodePhoneNumber(ctx context.Context, userId int32, countryCode string, phoneNumber string) (*dao_customer.Customer, error)
	GetByWAId(ctx context.Context, userId int32, waId string) (*dao_customer.Customer, error)
	GetByMetaUserId(ctx context.Context, userId int32, metaUserId string) (*dao_customer.Customer, error)
	GetByWAIdOrMetaUserId(ctx context.Context, userId int32, waId string, metaUserId string) (*dao_customer.Customer, error)
	ListByIds(ctx context.Context, userId int32, ids []int32) ([]dao_customer.Customer, error)
	CountByIds(ctx context.Context, userId int32, ids []int32) (int, error)
	CountActiveByUserId(ctx context.Context, userId int32) (int, error)
	CountActiveByMetaBusinessAccountId(ctx context.Context, metaBusinessAccountId string) (int, error)
	GetDistinctTags(ctx context.Context, userId int32) ([]string, error)
	List(ctx context.Context, userId int32, name string, phoneNumber string, order types.OrderCustomersType, status types.CustomerStatus, tags []string, page int, pageSize int) (customers []dao_customer.Customer, totalItems int, totalPages int, err error)
	GetByImportedPhoneNumber(ctx context.Context, userId int32, importedPhoneNumber string) (*dao_customer.Customer, error)
	GetByToken(ctx context.Context, token string) (*dao_customer.Customer, error)
}
