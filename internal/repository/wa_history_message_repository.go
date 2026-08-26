package repository

import dao_wa "github.com/jjcheng/wawa-go/internal/dao/wa"

type WAHistoryMessageRepository interface {
	Repository[dao_wa.HistoryMessage]
}
