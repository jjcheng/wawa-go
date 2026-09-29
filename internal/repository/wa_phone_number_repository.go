package repository

import (
	"context"

	dao_wa "github.com/jjcheng/wawa-go/internal/dao/wa"
	"github.com/jjcheng/wawa-go/internal/types"
)

type WAPhoneNumberRepository interface {
	Repository[dao_wa.PhoneNumber]
	CountByBusinessAccountId(ctx context.Context, businessAccountId int32) (int, error)
	// only used in embedded signup or incoming message, no user object
	GetByMetaPhoneNumberId(ctx context.Context, metaPhoneNumberId string) (*dao_wa.PhoneNumber, error)
	// used to list all phone numbers in user's meta business account
	ListByBusinessAccountId(ctx context.Context, businessAccountId int32, status types.WAPhoneNumberStatus, page int, pageSize int) (phoneNumbers []dao_wa.PhoneNumber, totalCount int, totalPages int, err error)
	// used when a master is updating a user
	GetBusinessPortfolioAndAccountByUserId(ctx context.Context, userId int32) (*dao_wa.BusinessPortfolio, *dao_wa.BusinessAccount, error)
	// used in authenticate middleware
	GetByUserId(ctx context.Context, userId int32) ([]dao_wa.PhoneNumber, *dao_wa.BusinessAccount, *dao_wa.BusinessPortfolio, error)
	ListUnassigned(ctx context.Context, businessAccountId int32) ([]dao_wa.PhoneNumber, error)
	// business agent
}
