package repository

import (
	"context"

	dao_wa "github.com/jjcheng/wawa-go/internal/dao/wa"
)

type WABusinessAgentKeywordRespository interface {
	Repository[dao_wa.BusinessAgentKeyword]
	ListByPhoneNumberId(ctx context.Context, phoneNumberId int32) ([]dao_wa.BusinessAgentKeyword, error)
	ListMatchingByPhoneNumberId(ctx context.Context, phoneNumberId int32, messageText string) ([]dao_wa.BusinessAgentKeyword, error)
}
