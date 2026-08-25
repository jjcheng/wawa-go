package dao_cm

import (
	"time"

	"github.com/jjcheng/wawa-go/internal/dao"
)

type RefreshToken struct {
	dao.DAOBase
	UserId        int32      `gorm:"column:user_id;not null"`
	TokenHash     string     `gorm:"column:token_hash;not null;unique"`
	TokenFamily   string     `gorm:"column:token_family;not null"`
	IssuedAt      time.Time  `gorm:"column:issued_at;not null"`
	ExpiresAt     time.Time  `gorm:"column:expires_at;not null"`
	LastUsedAt    *time.Time `gorm:"column:last_used_at"`
	UserAgent     *string    `gorm:"column:user_agent"`
	IPAddress     *string    `gorm:"column:ip_address"`
	IsRevoked     bool       `gorm:"column:is_revoked;not null;default:false"`
	RevokedAt     *time.Time `gorm:"column:revoked_at"`
	RevokedReason *string    `gorm:"column:revoked_reason"`
}

func (RefreshToken) TableName() string {
	return "cm_refresh_token"
}

func (rt RefreshToken) Base() dao.DAOBase {
	return rt.DAOBase
}

// IsExpired checks if the refresh token is expired
func (rt RefreshToken) IsExpired() bool {
	return time.Now().After(rt.ExpiresAt)
}

// IsValid checks if the token is valid (not expired, not revoked)
func (rt RefreshToken) IsValid() bool {
	return !rt.IsExpired() && !rt.IsRevoked
}
