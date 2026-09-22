package dto_account

import (
	"time"

	dao_account "github.com/jjcheng/wawa-go/internal/dao/account"
	"github.com/jjcheng/wawa-go/internal/dto"
)

type Session struct {
	dto.DTOBase
	ExpiresAt  time.Time  `json:"expires_at"`
	LastUsedAt time.Time  `json:"last_used_at"`
	RevokedAt  *time.Time `json:"revoked_at"`
	IP         string     `json:"ip"`
	UserAgent  string     `json:"user_agent"`
}

func NewSession(session dao_account.Session) Session {
	return Session{
		DTOBase: dto.DTOBase{
			Id:            session.Id,
			AddedAt:       session.AddedAt,
			LastUpdatedAt: session.LastUpdatedAt,
		},
		ExpiresAt:  session.ExpiresAt,
		RevokedAt:  session.RevokedAt,
		LastUsedAt: session.LastUsedAt,
		IP:         session.IP,
		UserAgent:  session.UserAgent,
	}
}
