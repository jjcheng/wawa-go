package repository

import (
	dao_wa "github.com/jjcheng/wawa-go/internal/dao/wa"
)

type WAMessageStatusEventRepository interface {
	Repository[dao_wa.MessageStatusEvent]
}
