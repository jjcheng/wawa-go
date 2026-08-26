package gormdb

import (
	dao_wa "github.com/jjcheng/wawa-go/internal/dao/wa"
	"github.com/jjcheng/wawa-go/internal/repository"
	"github.com/jjcheng/wawa-go/internal/service"
	"gorm.io/gorm"
)

type WAHistoryMessageRepository struct {
	repository.Repository[dao_wa.HistoryMessage]
}

func NewWAHistoryMessageRepository(db *gorm.DB, logger *service.Logger) repository.WAHistoryMessageRepository {
	return &WAHistoryMessageRepository{Repository: NewRepository[dao_wa.HistoryMessage](db, logger)}
}
