package dao_ai_agent

import (
	"github.com/jjcheng/wawa-go/internal/dao"
	"github.com/jjcheng/wawa-go/internal/types"
)

type FunctionParameter struct {
	dao.DAOBase
	FunctionId int32                              `gorm:"column:function_id"`
	Name       string                             `gorm:"column:name"`
	Type       types.AIAgentFunctionParameterType `gorm:"column:type"`
	Required   bool                               `gorm:"column:required"`
}

func (FunctionParameter) TableName() string {
	return "ai_agent.function_parameters"
}

func (functionParameter FunctionParameter) Base() dao.DAOBase {
	return functionParameter.DAOBase
}
