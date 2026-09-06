package repository

import (
	"context"

	dao_wa "github.com/jjcheng/wawa-go/internal/dao/wa"
)

type WAMessageRepository interface {
	Repository[dao_wa.Message]
	List(ctx context.Context, phoneNumberId string, customerMetaId string, customerPhoneNumber string) ([]dao_wa.Message, error)
}
