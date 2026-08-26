package dao_account

import (
	"time"

	"github.com/jjcheng/wawa-go/internal/dao"
)

type Session struct {
	dao.DAOBase
	UserId     int32      `gorm:"column:user_id"`
	TokenHash  string     `gorm:"column:token_hash"`
	ExpiresAt  time.Time  `gorm:"column:expires_at"`
	LastUsedAt time.Time  `gorm:"column:last_used_at"`
	RevokedAt  *time.Time `gorm:"column:revoked_at"`
}

func (Session) TableName() string {
	return "account.sessions"
}

func (session Session) Base() dao.DAOBase {
	return session.DAOBase
}
