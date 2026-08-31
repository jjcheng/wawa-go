package dto_account

import (
	"time"

	dao_account "github.com/jjcheng/wawa-go/internal/dao/account"
	"github.com/jjcheng/wawa-go/internal/dto"
)

type Session struct {
	dto.DTOBase
	UserId            int32      `json:"user_id"`
	AccessTokenHashed string     `json:"access_token_hashed"`
	ExpiresAt         time.Time  `json:"expires_at"`
	LastUsedAt        time.Time  `json:"last_used_at"`
	RevokedAt         *time.Time `json:"revoked_at"`
	IP                string     `json:"ip"`
	UserAgent         string     `json:"user_agent"`
}

func NewUserSession(session dao_account.Session) Session {
	return Session{
		DTOBase: dto.DTOBase{
			Id:         session.Id,
			EntryDate:  session.EntryDate,
			LastUpdate: session.LastUpdate,
		},
		UserId:            session.UserId,
		AccessTokenHashed: session.AccessTokenHashed,
		ExpiresAt:         session.ExpiresAt,
		RevokedAt:         session.RevokedAt,
		LastUsedAt:        session.LastUsedAt,
		IP:                session.IP,
		UserAgent:         session.UserAgent,
	}
}
