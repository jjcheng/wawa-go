package dao_ai_agent

import "github.com/jjcheng/wawa-go/internal/dao"

type Function struct {
	dao.DAOBase
	ConnectorId int32  `gorm:"column:connector_id"`
	Title       string `gorm:"column:title"`
	Description string `gorm:"column:description"`
}
