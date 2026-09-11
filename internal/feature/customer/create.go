package feature_customer

import (
	"context"
	"errors"
	"net/http"
	"strings"

	dao_customer "github.com/jjcheng/wawa-go/internal/dao/customer"
	"github.com/jjcheng/wawa-go/internal/dto"
	dto_account "github.com/jjcheng/wawa-go/internal/dto/account"
	dto_customer "github.com/jjcheng/wawa-go/internal/dto/customer"
	"github.com/jjcheng/wawa-go/internal/exception"
	"github.com/jjcheng/wawa-go/internal/feature"
	"github.com/jjcheng/wawa-go/internal/service"
	"github.com/jjcheng/wawa-go/internal/types"
	"gorm.io/gorm"
)

type Create struct {
	DisplayName string   `json:"display_name" val:"required" description:"customer display name"`
	CountryCode string   `json:"country_code" val:"required" description:"customer country code"`
	PhoneNumber string   `json:"phone_number" val:"required" description:"customer phone number"`
	MetaUserId  string   `json:"meta_user_id" description:"a string given by Meta"`
	WAId        string   `json:"wa_id" description:"given by Meta"`
	Tags        []string `json:"tags" description:"tags of the customer"`
	Remarks     string   `json:"remarks" description:"for your own reference"`
}

func (create *Create) Validate() []exception.InputException {
	create.DisplayName = strings.TrimSpace(create.DisplayName)
	create.CountryCode = strings.TrimSpace(create.CountryCode)
	create.PhoneNumber = strings.TrimSpace(create.PhoneNumber)
	create.PhoneNumber = strings.ReplaceAll(create.PhoneNumber, "+", "")
	create.PhoneNumber = strings.ReplaceAll(create.PhoneNumber, " ", "")
	create.PhoneNumber = strings.ReplaceAll(create.PhoneNumber, "-", "")
	create.MetaUserId = strings.TrimSpace(create.MetaUserId)
	create.WAId = strings.TrimSpace(create.WAId)
	create.Remarks = strings.TrimSpace(create.Remarks)
	if create.Tags == nil {
		// without this will have postgres error
		create.Tags = []string{}
	} else {
		for i := range create.Tags {
			create.Tags[i] = strings.TrimSpace(create.Tags[i])
		}
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
		if create.WAId == "" {
			errors = append(errors, exception.NewInputException("wa_id", "missing WA ID"))
		}
	}
	return errors
}

func (create Create) Handle(ctx context.Context, user *dto_account.User, dependencies *service.Dependencies) dto.Response[*dto_customer.Customer] {
	if inputErrors := create.Validate(); len(inputErrors) > 0 {
		return dto.NewInvalidInputResponse[*dto_customer.Customer](inputErrors)
	}
	// check existing
	if create.CountryCode != "" {
		existing, err := dependencies.UnitOfWork.CustomerRepository().GetByCountryCodePhoneNumber(ctx, user.Id, create.CountryCode, create.PhoneNumber)
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return dto.NewFailedResponse[*dto_customer.Customer](http.StatusInternalServerError, types.ExceptionMessageInternalServerError)
		}
		if existing != nil {
			return dto.NewFailedResponse[*dto_customer.Customer](http.StatusConflict, "customer already exists")
		}
	} else if create.MetaUserId != "" {
		existing, err := dependencies.UnitOfWork.CustomerRepository().GetByMetaUserId(ctx, user.Id, create.MetaUserId)
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return dto.NewFailedResponse[*dto_customer.Customer](http.StatusInternalServerError, types.ExceptionMessageInternalServerError)
		}
		if existing != nil {
			return dto.NewFailedResponse[*dto_customer.Customer](http.StatusConflict, "customer already exists")
		}
	}
	customer := dao_customer.Customer{
		DisplayName:         create.DisplayName,
		CountryCode:         create.CountryCode,
		PhoneNumber:         create.PhoneNumber,
		MetaUserId:          create.MetaUserId,
		Tags:                create.Tags,
		UserId:              user.Id,
		WAId:                create.WAId,
		Status:              types.CustomerStatusActive,
		Remarks:             create.Remarks,
		ImportedPhoneNumber: create.WAId,
	}
	// avoice non null error
	if customer.AdditionalData == nil {
		customer.AdditionalData = map[string]any{}
	}
	if err := dependencies.UnitOfWork.CustomerRepository().Insert(ctx, &customer); err != nil {
		return dto.NewFailedResponse[*dto_customer.Customer](http.StatusInternalServerError, types.ExceptionMessageInternalServerError)
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
