package dto_ai_agent

import (
	dao_ai_agent "github.com/jjcheng/wawa-go/internal/dao/ai_agent"
	"github.com/jjcheng/wawa-go/internal/dto"
	"github.com/lib/pq"
)

type Website struct {
	dto.DTOBase
	BusinessAccountId int32          `json:"business_account_id"`
	URL               string         `json:"url"`
	ExcludePatterns   pq.StringArray `json:"exclude_patterns"`
	IncludeSubdomains bool           `json:"include_subdomains"`
	JobId             string         `json:"job_id"`
	Status            string         `json:"status"`
}

func NewWebsite(website dao_ai_agent.Website) Website {
	return Website{
		DTOBase: dto.DTOBase{
			Id:            website.Id,
			AddedAt:       website.AddedAt,
			LastUpdatedAt: website.LastUpdatedAt,
		},
		BusinessAccountId: website.BusinessAccountId,
		URL:               website.URL,
		ExcludePatterns:   website.ExcludePatterns,
		IncludeSubdomains: website.IncludeSubdomains,
		JobId:             website.JobId,
		Status:            website.Status,
	}
}
