package feature_wa_user_phone_number

import (
	"context"
	"errors"
	"net/http"

	dao_wa "github.com/jjcheng/wawa-go/internal/dao/wa"
	"github.com/jjcheng/wawa-go/internal/dto"
	dto_account "github.com/jjcheng/wawa-go/internal/dto/account"
	dto_wa "github.com/jjcheng/wawa-go/internal/dto/wa"
	"github.com/jjcheng/wawa-go/internal/exception"
	"github.com/jjcheng/wawa-go/internal/feature"
	"github.com/jjcheng/wawa-go/internal/service"
	"github.com/jjcheng/wawa-go/internal/types"
	"gorm.io/gorm"
)

type List struct {
	Page     int `form:"page" description:"page number from 1"`
	PageSize int `form:"page_size" description:"page size, default 10"`
}

func (list *List) Validate() []exception.InputException {
	if list.Page <= 0 {
		list.Page = 1
	}
	if list.PageSize <= 0 {
		list.PageSize = 10
	}
	inputErrors := []exception.InputException{}
	if list.PageSize > 10 {
		inputErrors = append(inputErrors, exception.NewInputException("page_size", "page size must be between 1 and 10"))
	}
	return inputErrors
}

func (list List) Handle(ctx context.Context, user *dto_account.User, dependencies *service.Dependencies) dto.Response[*dto.ListResponse[dto_wa.PhoneNumber]] {
	if errors := list.Validate(); len(errors) > 0 {
		return dto.NewInvalidInputResponse[*dto.ListResponse[dto_wa.PhoneNumber]](errors)
	}
	if user == nil {
		return dto.NewFailedResponse[*dto.ListResponse[dto_wa.PhoneNumber]](http.StatusForbidden, "you are not authenticated")
	}
	var phoneNumbers []dao_wa.PhoneNumber
	var ex error
	var totalCount, totalPages int
	if user.Type == types.UserTypeMaster {
		_, businessAccount, err := dependencies.UnitOfWork.WAUserPhoneNumberRepository().GetBusinessPortfolioAndAccountByUserId(ctx, user.Id)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return dto.NewFailedResponse[*dto.ListResponse[dto_wa.PhoneNumber]](http.StatusUnauthorized, "you are not authorized to access a WhatsApp business portfolio")
			}
			return dto.NewFailedResponse[*dto.ListResponse[dto_wa.PhoneNumber]](http.StatusInternalServerError, types.ExceptionMessageInternalServerError)
		}
		phoneNumbers, totalCount, totalPages, ex = dependencies.UnitOfWork.WAPhoneNumberRepository().ListByMetaBusinessAccountId(ctx, businessAccount.MetaWABAId, list.Page, list.PageSize)
	} else {
		phoneNumbers, totalCount, totalPages, ex = dependencies.UnitOfWork.WAUserPhoneNumberRepository().ListPhoneNumbersByUserId(ctx, user.Id, list.Page, list.PageSize)
	}
	if ex != nil {
		return dto.NewFailedResponse[*dto.ListResponse[dto_wa.PhoneNumber]](http.StatusInternalServerError, types.ExceptionMessageInternalServerError)
	}
	items := make([]dto_wa.PhoneNumber, len(phoneNumbers))
	for index, phoneNumber := range phoneNumbers {
		items[index] = dto_wa.NewPhoneNumber(phoneNumber)
	}
	response := dto.NewPagedListResponse(items, totalPages, totalCount)
	return dto.NewSuccessResponse(&response)
}

func (List) APISettings() feature.APISettings {
	return feature.NewAPISettings(
		"List user WhatsApp phone numbers",
		"Lists WhatsApp phone numbers in pages: all numbers in a master's business portfolio, or numbers assigned to another authenticated user.",
		types.HttpRequestTypeQuery,
		http.MethodGet,
		"/v1/wa/user-phone-numbers",
		true,
		true,
		types.APITagWA,
		[]feature.APIError{
			feature.NewAPIError(*exception.NewCustomException("you are not authenticated", http.StatusForbidden)),
			feature.NewAPIError(*exception.NewCustomException("you are not authorized to access a WhatsApp business portfolio", http.StatusUnauthorized)),
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageInternalServerError, http.StatusInternalServerError)),
		},
	)
}
