package dto_account

import (
	dao_account "github.com/jjcheng/wawa-go/internal/dao/account"
	"github.com/jjcheng/wawa-go/internal/dto"
	dto_wa "github.com/jjcheng/wawa-go/internal/dto/wa"
	"github.com/jjcheng/wawa-go/internal/helper"
)

type UserPhoneNumber struct {
	dto.DTOBase
	UserId        int32               `json:"user_id"`
	PhoneNumberId int32               `json:"phone_number_id"`
	PhoneNumber   *dto_wa.PhoneNumber `json:"phone_number" title:"Phone number"`
}

func NewUserPhoneNumber(userPhoneNumber *dao_account.UserPhoneNumber) UserPhoneNumber {
	return UserPhoneNumber{
		DTOBase: dto.DTOBase{
			Id:            userPhoneNumber.Id,
			AddedAt:       userPhoneNumber.AddedAt,
			LastUpdatedAt: userPhoneNumber.LastUpdatedAt,
		},
		UserId:        userPhoneNumber.UserId,
		PhoneNumberId: userPhoneNumber.PhoneNumberId,
		PhoneNumber:   helper.ConvertToPointer(dto_wa.NewPhoneNumber(*userPhoneNumber.PhoneNumber)),
	}
}
