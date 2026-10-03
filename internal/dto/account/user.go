package dto_account

import (
	"time"

	dao_account "github.com/jjcheng/wawa-go/internal/dao/account"
	"github.com/jjcheng/wawa-go/internal/dto"
	dto_wa "github.com/jjcheng/wawa-go/internal/dto/wa"
	"github.com/jjcheng/wawa-go/internal/helper"
	"github.com/jjcheng/wawa-go/internal/types"
)

type User struct {
	dto.DTOBase
	Name              string           `json:"name" title:"Name"`
	CountryCode       string           `json:"country_code" title:"Country code"`
	PhoneNumber       string           `json:"phone_number" title:"Phone number"`
	Email             string           `json:"email" title:"Email"`
	Description       string           `json:"description"`
	Type              types.UserType   `json:"type" title:"Type"`
	Status            types.UserStatus `json:"status" title:"Status"`
	BusinessAccountId int32            `json:"business_account_id"`
	New               bool             `json:"-"`
	// got from session
	AccessToken       string     `json:"access_token,omitempty" description:"only returned during login"`
	AccessTokenExpiry *time.Time `json:"access_token_expiry,omitempty" description:"access token expiry time"`
	// to retrieve all WA related objects
	WA *UserWA `json:"-"`
	// login info
	Session *Session `json:"-"`
	// lazy loaded
	AssignedPhoneNumbers []AssignedPhoneNumber `json:"assigned_phone_numbers"`
}

type AssignedPhoneNumber struct {
	Id                 int32  `json:"id"`
	MetaPhoneNumberId  string `json:"meta_phone_number_id"`
	DisplayPhoneNumber string `json:"display_phone_number"`
	Name               string `json:"name"`
	MetaAgentId        string `json:"meta_agent_id"`
	AgentRunning       bool   `json:"agent_running"`
}

type UserWA struct {
	PhoneNumbers                 []dto_wa.PhoneNumber      `json:"-"`
	BusinessAccount              *dto_wa.BusinessAccount   `json:"-"`
	BusinessPortfolio            *dto_wa.BusinessPortfolio `json:"-"`
	BusinessPortfolioAccessToken string                    `json:"-"`
}

func (userWA *UserWA) PhoneNumberIds() []int32 {
	return helper.Map(userWA.PhoneNumbers, func(pn dto_wa.PhoneNumber) int32 {
		return pn.Id
	})
}

func (userWA *UserWA) PhoneNumberWAIds() []string {
	return helper.Map(userWA.PhoneNumbers, func(pn dto_wa.PhoneNumber) string {
		return pn.WAId
	})
}

func NewUser(user dao_account.User) User {
	d := User{
		DTOBase: dto.DTOBase{
			Id:            user.Id,
			AddedAt:       user.AddedAt,
			LastUpdatedAt: user.LastUpdatedAt,
		},
		Name:              user.Name,
		CountryCode:       user.CountryCode,
		PhoneNumber:       user.PhoneNumber,
		Email:             user.Email,
		Description:       user.Description,
		Type:              user.Type,
		Status:            user.Status,
		BusinessAccountId: user.BusinessAccountId,
	}
	return d
}
