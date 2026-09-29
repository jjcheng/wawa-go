package dto_wa

import (
	"fmt"
	"net/url"
	"strings"

	dao_wa "github.com/jjcheng/wawa-go/internal/dao/wa"
	"github.com/jjcheng/wawa-go/internal/dto"
	"github.com/jjcheng/wawa-go/internal/types"
)

type PhoneNumber struct {
	dto.DTOBase
	BusinessAccountId  int32                     `json:"business_account_id"`
	MetaPhoneNumberId  string                    `json:"meta_phone_number_id"`
	DisplayPhoneNumber string                    `json:"display_phone_number"`
	Name               string                    `json:"name"`
	WAId               string                    `json:"wa_id"`
	Status             types.WAPhoneNumberStatus `json:"status"`
	MetaAgentId        string                    `json:"meta_agent_id"`
	AgentRunning       bool                      `json:"agent_running"`
	// not stored
	New           bool           `json:"-"`
	AssignedUsers []AssignedUser `json:"assigned_users"`
}

// only used to consolidate assigned user in list phone numbers api, to avoide circular reference
type AssignedUser struct {
	Id   int32  `json:"id"`
	Name string `json:"name"`
}

func NewPhoneNumber(phoneNumber dao_wa.PhoneNumber) PhoneNumber {
	d := PhoneNumber{
		DTOBase: dto.DTOBase{
			Id:            phoneNumber.Id,
			AddedAt:       phoneNumber.AddedAt,
			LastUpdatedAt: phoneNumber.LastUpdatedAt,
		},
		BusinessAccountId:  phoneNumber.BusinessAccountId,
		MetaPhoneNumberId:  phoneNumber.MetaPhoneNumberId,
		DisplayPhoneNumber: phoneNumber.DisplayPhoneNumber,
		Name:               phoneNumber.Name,
		WAId:               phoneNumber.WAId,
		Status:             phoneNumber.Status,
		MetaAgentId:        phoneNumber.MetaAgentId,
		AgentRunning:       phoneNumber.AgentRunning,
	}
	return d
}

func (phoneNumber *PhoneNumber) WALink(text string) string {
	text = strings.TrimSpace(text)
	return fmt.Sprintf("https://wa.me/%s?text=%s", phoneNumber.WAId, url.QueryEscape(text))
}
