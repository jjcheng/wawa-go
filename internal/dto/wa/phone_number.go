package dto_wa

import (
	dao_wa "github.com/jjcheng/wawa-go/internal/dao/wa"
	"github.com/jjcheng/wawa-go/internal/dto"
)

type PhoneNumber struct {
	dto.DTOBase
	MetaBusinessPortfolioId string `json:"meta_business_portfolio_id"`
	MetaWABAId              string `json:"meta_waba_id"`
	MetaPhoneNumberId       string `json:"meta_phone_number_id"`
	PhoneNumber             string `json:"phone_number"`
	Name                    string `json:"name"`
	// lazy loaded
	BusinessPortfolio *BusinessPortfolio `json:"business_portfolio"`
	BusinessAccount   *BusinessAccount   `json:"business_account"`
}

func NewPhoneNumber(phoneNumber dao_wa.PhoneNumber) PhoneNumber {
	return PhoneNumber{
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
	}
}
