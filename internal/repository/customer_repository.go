package repository

import (
	"context"

	dao_customer "github.com/jjcheng/wawa-go/internal/dao/customer"
	"github.com/jjcheng/wawa-go/internal/types"
)

type CustomerRepository interface {
	Repository[dao_customer.Customer]
	GetByCountryCodePhoneNumber(ctx context.Context, userId int32, countryCode string, phoneNumber string) (*dao_customer.Customer, error)
	GetByMetaWAId(ctx context.Context, userId int32, metaWAId string) (*dao_customer.Customer, error)
	GetByMetaUserId(ctx context.Context, userId int32, metaUserId string) (*dao_customer.Customer, error)
	ListByIds(ctx context.Context, userId int32, ids []int32) ([]dao_customer.Customer, error)
	GetDistinctTags(ctx context.Context, userId int32) ([]string, error)
	List(ctx context.Context, userId int32, order types.OrderCustomersType, tags []string, page int, pageSize int) ([]dao_customer.Customer, error)
}
