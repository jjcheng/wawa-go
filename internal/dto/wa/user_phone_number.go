package dto_wa

import (
	dao_wa "github.com/jjcheng/wawa-go/internal/dao/wa"
	"github.com/jjcheng/wawa-go/internal/dto"
)

type UserPhoneNumber struct {
	dto.DTOBase
	UserId        int32 `json:"user_id"`
	PhoneNumberId int32 `json:"phone_number_id"`
}

func NewUserPhoneNumber(userPhoneNumber dao_wa.UserPhoneNumber) UserPhoneNumber {
	return UserPhoneNumber{
		DTOBase: dto.DTOBase{
			Id:         userPhoneNumber.Id,
			EntryDate:  userPhoneNumber.EntryDate,
			LastUpdate: userPhoneNumber.LastUpdate,
		},
		UserId:        userPhoneNumber.UserId,
		PhoneNumberId: userPhoneNumber.PhoneNumberId,
	}
}
