package repository

import (
	dao_account "github.com/jjcheng/wawa-go/internal/dao/account"
)

type AccountSettingRepository interface {
	Repository[dao_account.Setting]
}
