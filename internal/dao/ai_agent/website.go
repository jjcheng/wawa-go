package dao_ai_agent

import (
	"github.com/jjcheng/wawa-go/internal/dao"
	"github.com/lib/pq"
)

type Website struct {
	dao.DAOBase
	BusinessAccountId  int32          `gorm:"column:business_account_id"`
	URL                string         `gorm:"column:url"`
	ExcludeURLPatterns pq.StringArray `gorm:"column:exclude_url_patterns;type:text[]"`
}

func (Website) TableName() string {
	return "ai_agent.websites"
}

func (website Website) Base() dao.DAOBase {
	return website.DAOBase
}
