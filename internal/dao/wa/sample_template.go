package dao_wa

import (
	"github.com/jjcheng/wawa-go/internal/dao"
	"github.com/jjcheng/wawa-go/internal/types"
)

type SampleTemplate struct {
	dao.DAOBase
	Name            string                          `gorm:"column:name"`
	Category        types.WATemplateCategory        `gorm:"column:category"`
	Languauge       string                          `gorm:"column:language"`
	ParameterFormat types.WATemplateParameterFormat `gorm:"column:parameter_format"`
	Components      []map[string]any                `gorm:"column:components;type:jsonb;serializer:json"`
}

func (SampleTemplate) TableName() string {
	return "wa.sample_templates"
}

func (sampleTemplate SampleTemplate) Base() dao.DAOBase {
	return sampleTemplate.DAOBase
}
