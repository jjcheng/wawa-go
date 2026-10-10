package dto_ai_agent

import (
	dao_ai_agent "github.com/jjcheng/wawa-go/internal/dao/ai_agent"
	"github.com/jjcheng/wawa-go/internal/dto"
)

type Website struct {
	dto.DTOBase
	ProfileId         int32    `json:"profile_id"`
	URL               string   `json:"url"`
	ExcludePatterns   []string `json:"exclude_patterns"`
	IncludePatterns   []string `json:"include_patterns"`
	IncludeSubdomains bool     `json:"include_subdomains"`
	JobId             string   `json:"job_id"`
	Status            string   `json:"status"`
}

func NewWebsite(website dao_ai_agent.Website) Website {
	return Website{
		DTOBase: dto.DTOBase{
			Id:            website.Id,
			AddedAt:       website.AddedAt,
			LastUpdatedAt: website.LastUpdatedAt,
		},
		ProfileId:         website.ProfileId,
		URL:               website.URL,
		ExcludePatterns:   website.ExcludePatterns,
		IncludePatterns:   website.IncludePatterns,
		IncludeSubdomains: website.IncludeSubdomains,
		JobId:             website.JobId,
		Status:            website.Status,
	}
}
