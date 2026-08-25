package repository

import (
	"context"

	dao_account "github.com/jjcheng/wawa-go/internal/dao/account"
	"github.com/jjcheng/wawa-go/internal/exception"
)

type AccountUserRepository interface {
	Repository[dao_account.User]
	GetByEmailOrPhoneNumber(ctx context.Context, email string, phoneNumber string) (*dao_account.User, *exception.Exception)
	GetByPhoneNumber(ctx context.Context, phoneNumber string) (*dao_account.User, *exception.Exception)
	Get(ctx context.Context, id int32) (*dao_account.User, *exception.Exception)
	GetByOrganizationId(ctx context.Context, id int32, organizationId int32) (*dao_account.User, *exception.Exception)
}
