package feature_wa_phone_number

import (
	"context"
	"net/http"

	dao_account "github.com/jjcheng/wawa-go/internal/dao/account"
	"github.com/jjcheng/wawa-go/internal/dto"
	dto_account "github.com/jjcheng/wawa-go/internal/dto/account"
	dto_wa "github.com/jjcheng/wawa-go/internal/dto/wa"
	"github.com/jjcheng/wawa-go/internal/exception"
	"github.com/jjcheng/wawa-go/internal/feature"
	"github.com/jjcheng/wawa-go/internal/helper"
	"github.com/jjcheng/wawa-go/internal/service"
	"github.com/jjcheng/wawa-go/internal/types"
)

type List struct {
	Status   types.WAPhoneNumberStatus `form:"status" description:"status of the phone number"`
	Page     int                       `form:"page" description:"page number from 1"`
	PageSize int                       `form:"page_size" description:"page size, default 25"`
}

func (list *List) Validate() []exception.InputException {
	if list.Page <= 0 {
		list.Page = 1
	}
	if list.PageSize <= 0 {
		list.PageSize = 25
	}
	return nil
}

func (list List) Handle(ctx context.Context, user *dto_account.User, dependencies *service.Dependencies) dto.Response[*dto.ListResponse[dto_wa.PhoneNumber]] {
	if user == nil {
		return dto.NewFailedResponse[*dto.ListResponse[dto_wa.PhoneNumber]](http.StatusForbidden, types.ExceptionMessageForbidden, nil)
	}
	if user.WA == nil {
		return dto.NewFailedResponse[*dto.ListResponse[dto_wa.PhoneNumber]](http.StatusUnauthorized, types.ExceptionMessageUnauthorized, nil)
	}
	if inputErrors := list.Validate(); len(inputErrors) > 0 {
		return dto.NewInvalidInputResponse[*dto.ListResponse[dto_wa.PhoneNumber]](inputErrors)
	}
	items := make([]dto_wa.PhoneNumber, 0)
	var totalCount, totalPages int
	if user.Type == types.UserTypeMaster {
		phoneNumbers, count, pages, err := dependencies.UnitOfWork.WAPhoneNumberRepository().ListByBusinessAccountId(ctx, user.BusinessAccountId, list.Status, list.Page, list.PageSize)
		if err != nil {
			return dto.NewFailedResponse[*dto.ListResponse[dto_wa.PhoneNumber]](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
		}
		totalCount = count
		totalPages = pages
		items = make([]dto_wa.PhoneNumber, len(phoneNumbers))
		for index, phoneNumber := range phoneNumbers {
			items[index] = dto_wa.NewPhoneNumber(phoneNumber)
		}
		// bind assigned users count
		if len(items) > 0 {
			phoneNumberIds := helper.Map(items, func(pn dto_wa.PhoneNumber) int32 {
				return pn.Id
			})
			userPhoneNumbers, err := dependencies.UnitOfWork.AccountUserPhoneNumberRepository().ListByPhoneNumberIds(ctx, phoneNumberIds)
			if err != nil {
				return dto.NewFailedResponse[*dto.ListResponse[dto_wa.PhoneNumber]](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
			}
			for i := range items {
				userPhoneNumbers := helper.Filter(userPhoneNumbers, func(au dao_account.UserPhoneNumber) bool {
					return au.PhoneNumberId == items[i].Id
				})
				var assignedUsers []dto_wa.AssignedUser
				for _, userPhoneNumber := range userPhoneNumbers {
					assignedUsers = append(assignedUsers, dto_wa.AssignedUser{
						Id:   userPhoneNumber.UserId,
						Name: userPhoneNumber.User.Name,
					})
				}
				items[i].AssignedUsers = assignedUsers
			}
		}
	} else {
		phoneNumbers := user.WA.PhoneNumbers
		if list.Status != "" {
			phoneNumbers = helper.Filter(phoneNumbers, func(pn dto_wa.PhoneNumber) bool {
				return pn.Status == list.Status
			})
		}
		items = phoneNumbers
		totalCount = 1
		totalPages = 1
	}
	response := dto.NewPagedListResponse(items, totalPages, totalCount)
	return dto.NewSuccessResponse(&response)
}

func (List) APISettings() feature.APISettings {
	return feature.NewAPISettings(
		"List WhatsApp phone numbers",
		"List WhatsApp phone numbers in entire business account if user is MASTER, or numbers assigned to current user.",
		types.HttpRequestTypeQuery,
		http.MethodGet,
		"/v1/wa/phone-numbers",
		true,
		true,
		types.APITagWA,
		[]feature.APIError{
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageForbidden, http.StatusForbidden)),
			feature.NewAPIError(*exception.NewCustomException("you are not authorized to access a WhatsApp business portfolio", http.StatusUnauthorized)),
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageInternalServerError, http.StatusInternalServerError)),
		},
	)
}
