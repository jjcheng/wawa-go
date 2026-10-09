package dao_ai_agent

import (
	"github.com/jjcheng/wawa-go/internal/dao"
	"github.com/lib/pq"
)

type Website struct {
	dao.DAOBase
	BusinessAccountId int32          `gorm:"column:business_account_id"`
	URL               string         `gorm:"column:url"`
	ExcludePatterns   pq.StringArray `gorm:"column:exclude_patterns;type:text[]"`
	IncludeSubdomains bool           `gorm:"column:include_subdomains"`
	JobId             string         `gorm:"column:job_id"`
	Status            string         `gorm:"column:status"`
}

func (Website) TableName() string {
	return "ai_agent.websites"
}

func (website Website) Base() dao.DAOBase {
	return website.DAOBase
}
