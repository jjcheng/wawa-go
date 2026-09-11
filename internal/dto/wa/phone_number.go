package dto_wa

import (
	dao_wa "github.com/jjcheng/wawa-go/internal/dao/wa"
	"github.com/jjcheng/wawa-go/internal/dto"
	"github.com/jjcheng/wawa-go/internal/types"
)

type PhoneNumber struct {
	dto.DTOBase
	MetaBusinessPortfolioId string                    `json:"meta_business_portfolio_id"`
	MetaWABAId              string                    `json:"meta_waba_id"`
	MetaPhoneNumberId       string                    `json:"meta_phone_number_id"`
	PhoneNumber             string                    `json:"phone_number"`
	Name                    string                    `json:"name"`
	UserId                  int32                     `json:"user_id"`
	WAId                    string                    `json:"wa_id"`
	Status                  types.WAPhoneNumberStatus `json:"status"`
	UserName                string                    `json:"user_name"`
}

func NewPhoneNumber(phoneNumber dao_wa.PhoneNumber) PhoneNumber {
	d := PhoneNumber{
		DTOBase: dto.DTOBase{
			Id:         phoneNumber.Id,
			EntryDate:  phoneNumber.EntryDate,
			LastUpdate: phoneNumber.LastUpdate,
		},
		MetaBusinessPortfolioId: phoneNumber.MetaBusinessPortfolioId,
		MetaWABAId:              phoneNumber.MetaWABAId,
		MetaPhoneNumberId:       phoneNumber.MetaPhoneNumberId,
		PhoneNumber:             phoneNumber.PhoneNumber,
		Name:                    phoneNumber.Name,
		WAId:                    phoneNumber.WAId,
		UserId:                  phoneNumber.UserId,
		Status:                  phoneNumber.Status,
		UserName:                phoneNumber.UserName,
	}
	return d
}
