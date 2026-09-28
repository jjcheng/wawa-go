package feature_customer

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/google/uuid"
	dao_customer "github.com/jjcheng/wawa-go/internal/dao/customer"
	"github.com/jjcheng/wawa-go/internal/dto"
	dto_account "github.com/jjcheng/wawa-go/internal/dto/account"
	dto_customer "github.com/jjcheng/wawa-go/internal/dto/customer"
	dto_wa "github.com/jjcheng/wawa-go/internal/dto/wa"
	"github.com/jjcheng/wawa-go/internal/exception"
	"github.com/jjcheng/wawa-go/internal/feature"
	"github.com/jjcheng/wawa-go/internal/helper"
	"github.com/jjcheng/wawa-go/internal/service"
	"github.com/jjcheng/wawa-go/internal/types"
	"gorm.io/gorm"
)

type Create struct {
	DisplayName   string   `json:"display_name" val:"required" description:"display name given by user"`
	WADisplayName string   `json:"wa_display_name" description:"display name given by WhatsApp"`
	CountryCode   string   `json:"country_code" val:"required" description:"customer country code"`
	PhoneNumber   string   `json:"phone_number" val:"required" description:"customer phone number"`
	MetaUserId    string   `json:"meta_user_id" description:"a string given by Meta"`
	Tags          []string `json:"tags" description:"tags of the customer"`
	Remarks       string   `json:"remarks" description:"for your own reference"`
	WAId          string   `json:"wa_id" description:"optional waId from incoming messages"`
	PhoneNumberId int32    `json:"phone_number_id" val:"required" description:"which phone number to assign this customer to"`
	FromIncoming  bool     `json:"-"` // if from incoming WA message, there is no user in Handle()
}

func (create *Create) Validate() []exception.InputException {
	create.DisplayName = strings.TrimSpace(create.DisplayName)
	create.WADisplayName = strings.TrimSpace(create.WADisplayName)
	create.CountryCode = strings.TrimSpace(create.CountryCode)
	create.PhoneNumber = strings.TrimSpace(create.PhoneNumber)
	create.PhoneNumber = strings.ReplaceAll(create.PhoneNumber, "+", "")
	create.PhoneNumber = strings.ReplaceAll(create.PhoneNumber, " ", "")
	create.PhoneNumber = strings.ReplaceAll(create.PhoneNumber, "-", "")
	create.MetaUserId = strings.TrimSpace(create.MetaUserId)
	create.WAId = strings.TrimSpace(create.WAId)
	create.Remarks = strings.TrimSpace(create.Remarks)
	for i := range create.Tags {
		create.Tags[i] = strings.TrimSpace(create.Tags[i])
	}
	errors := []exception.InputException{}
	if create.DisplayName == "" {
		errors = append(errors, exception.NewInputException("display_name", "missing display name"))
	}
	// if metaUserId is empty, check for country code + phone number
	if create.MetaUserId == "" {
		if create.CountryCode == "" {
			errors = append(errors, exception.NewInputException("country_code", "missing country code"))
		}
		if create.PhoneNumber == "" {
			errors = append(errors, exception.NewInputException("phone_number", "missing phone number"))
		}
	}
	if create.PhoneNumberId <= 0 {
		errors = append(errors, exception.NewInputException("phone_number_id", "missing phone number id"))
	}
	return errors
}

func (create Create) Handle(ctx context.Context, user *dto_account.User, dependencies *service.Dependencies) dto.Response[*dto_customer.Customer] {
	if !create.FromIncoming {
		if user == nil {
			return dto.NewFailedResponse[*dto_customer.Customer](http.StatusForbidden, types.ExceptionMessageForbidden, nil)
		}
		if user.WA == nil {
			return dto.NewFailedResponse[*dto_customer.Customer](http.StatusUnauthorized, types.ExceptionMessageUnauthorized, nil)
		}
		if !helper.Any(user.WA.PhoneNumbers, func(pn dto_wa.PhoneNumber) bool {
			return pn.Id == create.PhoneNumberId
		}) {
			return dto.NewFailedResponse[*dto_customer.Customer](http.StatusUnauthorized, types.ExceptionMessageUnauthorized, nil)
		}
	}
	if inputErrors := create.Validate(); len(inputErrors) > 0 {
		return dto.NewInvalidInputResponse[*dto_customer.Customer](inputErrors)
	}
	// check existing
	if create.CountryCode != "" {
		existing, err := dependencies.UnitOfWork.CustomerRepository().GetByCountryCodePhoneNumber(ctx, create.PhoneNumberId, create.CountryCode, create.PhoneNumber)
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return dto.NewFailedResponse[*dto_customer.Customer](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
		}
		if existing != nil {
			return dto.NewFailedResponse[*dto_customer.Customer](http.StatusConflict, "customer already exists", nil)
		}
	} else if create.MetaUserId != "" { // created by incoming message with only meta user id
		existing, err := dependencies.UnitOfWork.CustomerRepository().GetByMetaUserId(ctx, create.PhoneNumberId, create.MetaUserId)
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return dto.NewFailedResponse[*dto_customer.Customer](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
		}
		if existing != nil {
			return dto.NewFailedResponse[*dto_customer.Customer](http.StatusConflict, "customer already exists", nil)
		}
	}
	// if a WAId already provided, use it, otherwise derive from countryCode+phoneNumber
	waId := create.WAId
	if waId == "" {
		waId = helper.GetWAId(create.CountryCode, create.PhoneNumber)
	}
	// insert
	customer := dao_customer.Customer{
		DisplayName:         create.DisplayName,
		WADisplayName:       create.WADisplayName,
		CountryCode:         create.CountryCode,
		PhoneNumber:         create.PhoneNumber,
		MetaUserId:          create.MetaUserId,
		Tags:                create.Tags,
		PhoneNumberId:       create.PhoneNumberId,
		WAId:                waId,
		Status:              types.CustomerStatusActive,
		Remarks:             create.Remarks,
		ImportedPhoneNumber: waId,
		Token:               uuid.NewString(),
		FromIncomingMessage: create.FromIncoming,
	}
	if err := dependencies.UnitOfWork.CustomerRepository().Insert(ctx, &customer); err != nil {
		return dto.NewFailedResponse[*dto_customer.Customer](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
	}
	result := dto_customer.NewCustomer(customer)
	return dto.NewSuccessResponse(&result)
}

func (Create) APISettings() feature.APISettings {
	return feature.NewAPISettings(
		"Create customer",
		"Creates a customer record.",
		types.HttpRequestTypeJSON,
		http.MethodPost,
		"/v1/customers",
		true,
		true,
		types.APITagCustomer,
		[]feature.APIError{
			feature.NewAPIError(*exception.NewCustomException("customer already exists", http.StatusConflict)),
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageInternalServerError, http.StatusInternalServerError)),
		},
	)
}
