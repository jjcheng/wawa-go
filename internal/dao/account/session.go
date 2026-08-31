package dao_account

import (
	"time"

	"github.com/jjcheng/wawa-go/internal/dao"
)

type Session struct {
	dao.DAOBase
	UserId            int32      `gorm:"column:user_id"`
	AccessTokenHashed string     `gorm:"column:access_token_hashed"`
	ExpiresAt         time.Time  `gorm:"column:expires_at"`
	LastUsedAt        time.Time  `gorm:"column:last_used_at"`
	RevokedAt         *time.Time `gorm:"column:revoked_at"`
	IP                string     `gorm:"column:ip"`
	UserAgent         string     `gorm:"column:user_agent"`
}

func (Session) TableName() string {
	return "account.sessions"
}

func (session Session) Base() dao.DAOBase {
	return session.DAOBase
}
