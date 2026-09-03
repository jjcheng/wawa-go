package repository

import (
	dao_wa "github.com/jjcheng/wawa-go/internal/dao/wa"
	"github.com/jjcheng/wawa-go/internal/types"
)

type WASampleTemplateRepository interface {
	Repository[dao_wa.SampleTemplate]
	List(category types.WATemplateCategory, language string) ([]dao_wa.SampleTemplate, error)
}
