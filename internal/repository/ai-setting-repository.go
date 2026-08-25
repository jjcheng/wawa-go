package repository

import (
	"context"

	dao_account "github.com/jjcheng/wawa-go/internal/dao/account"
	"github.com/jjcheng/wawa-go/internal/exception"
)

type AccountSettingRepository interface {
	Repository[dao_account.Setting]
	ListByOrganizationId(ctx context.Context, organizationId int32) ([]dao_account.Setting, *exception.Exception)
	GetByOrganizationId(ctx context.Context, id int32, organizationId int32) (*dao_account.Setting, *exception.Exception)
	GetByNameAndOrganizationId(ctx context.Context, name string, organizationId int32) (*dao_account.Setting, *exception.Exception)
}
