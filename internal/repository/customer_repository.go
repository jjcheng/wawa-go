package repository

import (
	"context"

	dao_customer "github.com/jjcheng/wawa-go/internal/dao/customer"
	"github.com/jjcheng/wawa-go/internal/types"
)

type CustomerRepository interface {
	Repository[dao_customer.Customer]
	GetByCountryCodePhoneNumber(ctx context.Context, phoneNumberId int32, countryCode string, phoneNumber string) (*dao_customer.Customer, error)
	GetByMetaUserId(ctx context.Context, phoneNumberId int32, metaUserId string) (*dao_customer.Customer, error)
	GetByWAIdOrMetaUserId(ctx context.Context, phoneNumberId int32, waId string, metaUserId string) (*dao_customer.Customer, error)
	ListByPhoneNumberIdsAndIds(ctx context.Context, phoneNumberIds []int32, ids []int32) ([]dao_customer.Customer, error)
	CountActiveByPhoneNumberIds(ctx context.Context, phoneNumberIds []int32) (int, error)
	CountActiveByBusinessAccountId(ctx context.Context, businessAccountId int32) (int, error)
	GetDistinctTagsByPhoneNumberIds(ctx context.Context, phoneNumberIds []int32) ([]string, error)
	List(ctx context.Context, phoneNumberIds []int32, name string, order types.OrderCustomersType, status types.CustomerStatus, tags []string, page int, pageSize int) (customers []dao_customer.Customer, totalItems int, totalPages int, err error)
	GetByImportedPhoneNumber(ctx context.Context, phoneNumberId int32, importedPhoneNumber string) (*dao_customer.Customer, error)
	DeleteByIds(ctx context.Context, ids []int32) error
	UpdateStatusByIds(ctx context.Context, ids []int32, status types.CustomerStatus) error
}
