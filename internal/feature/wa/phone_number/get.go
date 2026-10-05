package feature_wa_phone_number

import (
	"context"
	"errors"
	"net/http"

	"github.com/jjcheng/wawa-go/internal/dto"
	dto_account "github.com/jjcheng/wawa-go/internal/dto/account"
	dto_wa "github.com/jjcheng/wawa-go/internal/dto/wa"
	"github.com/jjcheng/wawa-go/internal/exception"
	"github.com/jjcheng/wawa-go/internal/feature"
	"github.com/jjcheng/wawa-go/internal/helper"
	"github.com/jjcheng/wawa-go/internal/service"
	"github.com/jjcheng/wawa-go/internal/types"
	"gorm.io/gorm"
)

type Get struct {
	Id int32 `uri:"phone_number_id" description:"id of the phone number"`
}

func (get *Get) Validate() []exception.InputException {
	return nil
}

func (get Get) Handle(ctx context.Context, user *dto_account.User, dependencies *service.Dependencies) dto.Response[*service.WhatsAppPhoneNumberDetailsResponse] {
	if user == nil {
		return dto.NewFailedResponse[*service.WhatsAppPhoneNumberDetailsResponse](http.StatusForbidden, types.ExceptionMessageForbidden, nil)
	}
	if user.WA == nil || user.WA.BusinessPortfolioAccessToken == "" {
		return dto.NewFailedResponse[*service.WhatsAppPhoneNumberDetailsResponse](http.StatusUnauthorized, types.ExceptionMessageUnauthorized, nil)
	}
	if inputErrors := get.Validate(); len(inputErrors) > 0 {
		return dto.NewInvalidInputResponse[*service.WhatsAppPhoneNumberDetailsResponse](inputErrors)
	}
	if user.Type != types.UserTypeMaster {
		if !helper.Any(user.WA.PhoneNumbers, func(pn dto_wa.PhoneNumber) bool {
			return pn.Id == get.Id
		}) {
			return dto.NewFailedResponse[*service.WhatsAppPhoneNumberDetailsResponse](http.StatusUnauthorized, types.ExceptionMessageUnauthorized, nil)
		}
	}
	storedPhoneNumber, err := dependencies.UnitOfWork.WAPhoneNumberRepository().GetById(ctx, get.Id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return dto.NewFailedResponse[*service.WhatsAppPhoneNumberDetailsResponse](http.StatusNotFound, "phone number not found", nil)
		}
		return dto.NewFailedResponse[*service.WhatsAppPhoneNumberDetailsResponse](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
	}
	// if user is master, it must be belong to business account; non-master already check earlier
	if storedPhoneNumber.BusinessAccountId != user.BusinessAccountId {
		return dto.NewFailedResponse[*service.WhatsAppPhoneNumberDetailsResponse](http.StatusUnauthorized, types.ExceptionMessageUnauthorized, nil)
	}
	// get phone number from meta
	metaPhoneNumber, err := dependencies.Whatsapp.GetPhoneNumber(ctx, storedPhoneNumber.MetaPhoneNumberId, user.WA.BusinessPortfolioAccessToken)
	if err != nil {
		return dto.NewFailedResponse[*service.WhatsAppPhoneNumberDetailsResponse](http.StatusBadGateway, err.Error(), err)
	}
	// update status in db if needed
	var newStatus types.WAPhoneNumberStatus
	if metaPhoneNumber.Status == "CONNECTED" && storedPhoneNumber.Status != types.WAPhoneNumberStatusConnected {
		newStatus = types.WAPhoneNumberStatusConnected
	} else if metaPhoneNumber.Status != "CONNECTED" && storedPhoneNumber.Status == types.WAPhoneNumberStatusConnected {
		newStatus = types.WAPhoneNumberStatusDisconnected
	}
	if newStatus != "" {
		storedPhoneNumber.Status = newStatus
		// ignore any error
		_ = dependencies.UnitOfWork.WAPhoneNumberRepository().Update(ctx, storedPhoneNumber)
	}
	// get assigned users
	assignedUsers, err := dependencies.UnitOfWork.AccountUserPhoneNumberRepository().ListByPhoneNumberId(ctx, storedPhoneNumber.Id)
	if err != nil {
		return dto.NewFailedResponse[*service.WhatsAppPhoneNumberDetailsResponse](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
	}
	for _, assignedUser := range assignedUsers {
		metaPhoneNumber.AssignedUsers = append(metaPhoneNumber.AssignedUsers, dto_wa.AssignedUser{
			Id:   assignedUser.UserId,
			Name: assignedUser.User.Name,
		})
	}
	return dto.NewSuccessResponse(metaPhoneNumber)
}

func (Get) APISettings() feature.APISettings {
	return feature.NewAPISettings(
		"Get WhatsApp phone number",
		"Get the current WhatsApp phone number status and configuration by WhatsApp API.",
		types.HttpRequestTypeUri,
		http.MethodGet,
		"/v1/wa/phone-numbers/:phone_number_id",
		true,
		true,
		types.APITagWA,
		[]feature.APIError{
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageForbidden, http.StatusForbidden)),
			feature.NewAPIError(*exception.NewCustomException("you are not authorized to access this WhatsApp phone number", http.StatusUnauthorized)),
			feature.NewAPIError(*exception.NewCustomException("phone number not found", http.StatusNotFound)),
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageBadGateway, http.StatusBadGateway)),
		},
	)
}
