package repository

import (
	"context"

	dao_wa "github.com/jjcheng/wawa-go/internal/dao/wa"
	"github.com/jjcheng/wawa-go/internal/types"
)

type WAPhoneNumberRepository interface {
	Repository[dao_wa.PhoneNumber]
	Get(ctx context.Context, id int32) (*dao_wa.PhoneNumber, error)
	CountByMetaBusinessAccountId(ctx context.Context, metaBusinessAccountId string) (int, error)
	// only used in embedded signup or incoming message, no user object
	GetByPhoneNumberId(ctx context.Context, phoneNumberId string) (*dao_wa.PhoneNumber, error)
	// used to list all phone numbers in user's meta business account
	ListByMetaBusinessAccountId(ctx context.Context, metaBusinessAccountId string, status types.WAPhoneNumberStatus, page int, pageSize int) (phoneNumbers []dao_wa.PhoneNumber, totalCount int, totalPages int, err error)
	// used when a master is updating a user
	GetBusinessPortfolioAndAccountByUserId(ctx context.Context, userId int32) (*dao_wa.BusinessPortfolio, *dao_wa.BusinessAccount, error)
	// used in authenticate middleware
	GetByUserId(ctx context.Context, userId int32) (*dao_wa.PhoneNumber, *dao_wa.BusinessAccount, *dao_wa.BusinessPortfolio, error)
}
