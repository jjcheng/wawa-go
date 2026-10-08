package dao_ai_agent

import (
	"github.com/jjcheng/wawa-go/internal/dao"
	"github.com/jjcheng/wawa-go/internal/types"
)

type Connector struct {
	dao.DAOBase
	ProfileId          int32                              `gorm:"column:profile_id"`
	BaseURL            string                             `gorm:"column:base_url"`
	AuthLocation       types.AIAgentConnectorAuthLocation `gorm:"column:auth_location"`
	AuthKey            string                             `gorm:"column:auth_key"`
	AuthValueEncrypted string                             `gorm:"column:auth_value_encrypted"`
	AuthValue          string                             `gorm:"-"` // not in db
}

func (Connector) TableName() string {
	return "ai_agent.connectors"
}

func (connector Connector) Base() dao.DAOBase {
	return connector.DAOBase
}
