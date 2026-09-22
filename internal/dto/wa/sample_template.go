package dto_wa

import (
	"encoding/json"

	dao_wa "github.com/jjcheng/wawa-go/internal/dao/wa"
	"github.com/jjcheng/wawa-go/internal/dto"
)

type SampleTemplate struct {
	dto.DTOBase
	TemplateBase
}

func NewSampleTemplate(sampleTemplate dao_wa.SampleTemplate) (*SampleTemplate, error) {
	d := SampleTemplate{
		DTOBase: dto.DTOBase{
			Id:            sampleTemplate.Id,
			AddedAt:       sampleTemplate.AddedAt,
			LastUpdatedAt: sampleTemplate.LastUpdatedAt,
		},
		TemplateBase: TemplateBase{
			Name:            sampleTemplate.Name,
			Language:        sampleTemplate.Languauge,
			Category:        sampleTemplate.Category,
			ParameterFormat: sampleTemplate.ParameterFormat,
		},
	}
	if len(sampleTemplate.Components) > 0 {
		payload, err := json.Marshal(sampleTemplate.Components)
		if err != nil {
			return nil, err
		}
		if err := json.Unmarshal(payload, &d.Components); err != nil {
			return nil, err
		}
		d.TemplateBase.Components = d.Components
		template := Template{TemplateBase: d.TemplateBase}
		d.TemplateBase.PreviewHTML = template.HTML(true, false)
		d.TemplateBase.RawHTML = template.HTML(false, false)
		d.TemplateBase.PreviewDarkHTML = template.HTML(true, true)
		d.TemplateBase.RawDarkHTML = template.HTML(false, true)
		d.TemplateBase.SendComponents = template.GetSendComponents()
	}
	return &d, nil
}
