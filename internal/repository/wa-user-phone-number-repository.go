package repository

import (
	dao_wa "github.com/jjcheng/wawa-go/internal/dao/wa"
)

type WAUserPhoneNumberRepository interface {
	Repository[dao_wa.UserPhoneNumber]
}
