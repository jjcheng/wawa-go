package feature_customer

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
	dao_customer "github.com/jjcheng/wawa-go/internal/dao/customer"
	"github.com/jjcheng/wawa-go/internal/dto"
	dto_account "github.com/jjcheng/wawa-go/internal/dto/account"
	"github.com/jjcheng/wawa-go/internal/exception"
	"github.com/jjcheng/wawa-go/internal/feature"
	"github.com/jjcheng/wawa-go/internal/helper"
	"github.com/jjcheng/wawa-go/internal/service"
	"github.com/jjcheng/wawa-go/internal/types"
	"gorm.io/gorm"
)

type Import struct {
	Contacts []Contact `json:"contacts" val:"required" description:"contacts to import"`
}

type Contact struct {
	DisplayName         string `json:"display_name"`
	PhoneNumber         string `json:"phone_number"`
	Email               string `json:"email,omitempty"`
	Address             string `json:"address,omitempty"`
	Organization        string `json:"organization,omitempty"`
	JobTitle            string `json:"job_title,omitempty"`
	Birthday            string `json:"birthday,omitempty"`
	Anniversary         string `json:"anniversary,omitempty"`
	Gender              string `json:"gender,omitempty"`
	TimeZone            string `json:"time_zone,omitempty"`
	Categories          string `json:"categories,omitempty"`
	Note                string `json:"note,omitempty"`
	Photo               string `json:"photo,omitempty"`
	URL                 string `json:"url,omitempty"`
	Skipped             bool   `json:"skipped"`
	SkipReason          string `json:"skip_reason"`
	ImportedPhoneNumber string `json:"-"` // to store original phone number
}

type ImportResult struct {
	ImportedCount int       `json:"imported_count"`
	Skipped       []Contact `json:"skipped"`
	Message       string    `json:"message"`
}

func (importCustomers *Import) Validate() []exception.InputException {
	if importCustomers == nil || len(importCustomers.Contacts) == 0 {
		return []exception.InputException{exception.NewInputException("contacts", "missing contacts")}
	}
	seen := map[string]struct{}{}
	for index := range importCustomers.Contacts {
		contact := &importCustomers.Contacts[index]
		contact.DisplayName = strings.TrimSpace(contact.DisplayName)
		contact.ImportedPhoneNumber = contact.PhoneNumber
		contact.PhoneNumber = helper.NormalizeWAId(contact.PhoneNumber)
		if contact.DisplayName == "" {
			contact.Skipped = true
			contact.SkipReason = "missing display name"
			continue
		}
		if contact.PhoneNumber == "" {
			contact.Skipped = true
			contact.SkipReason = "missing phone number"
			continue
		}
		if _, exists := seen[contact.PhoneNumber]; exists {
			contact.Skipped = true
			contact.SkipReason = "duplicate phone number"
			continue
		}
		seen[contact.PhoneNumber] = struct{}{}
	}
	return nil
}

func (importCustomers Import) Handle(ctx context.Context, user *dto_account.User, dependencies *service.Dependencies) dto.Response[*ImportResult] {
	if user == nil {
		return dto.NewFailedResponse[*ImportResult](http.StatusForbidden, "you are not authenticated")
	}
	if inputErrors := importCustomers.Validate(); len(inputErrors) > 0 {
		return dto.NewInvalidInputResponse[*ImportResult](inputErrors)
	}
	transaction := dependencies.UnitOfWork.BeginTransaction()
	committed := false
	defer func() {
		if !committed {
			transaction.Rollback()
		}
	}()
	customers := make([]dao_customer.Customer, 0, len(importCustomers.Contacts))
	result := &ImportResult{Skipped: []Contact{}}
	for _, contact := range importCustomers.Contacts {
		if contact.Skipped {
			result.Skipped = append(result.Skipped, contact)
			continue
		}
		countryCode, phoneNumber := parsePhoneNumber(contact.PhoneNumber)
		existing, err := transaction.CustomerRepository().GetByImportedPhoneNumber(ctx, user.Id, contact.ImportedPhoneNumber)
		if err != nil {
			if !errors.Is(err, gorm.ErrRecordNotFound) {
				return dto.NewFailedResponse[*ImportResult](http.StatusInternalServerError, types.ExceptionMessageInternalServerError)
			}
		}
		if existing != nil {
			contact.Skipped = true
			contact.SkipReason = "phone number already imported"
			result.Skipped = append(result.Skipped, contact)
			continue
		}
		customer := dao_customer.Customer{
			UserId:              user.Id,
			DisplayName:         contact.DisplayName,
			CountryCode:         countryCode,
			PhoneNumber:         phoneNumber,
			Tags:                contact.Tags(),
			Status:              types.CustomerStatusActive,
			Remarks:             contact.Note,
			AdditionalData:      contact.AdditionalData(),
			ImportedPhoneNumber: contact.ImportedPhoneNumber,
			Token:               strings.ReplaceAll(uuid.NewString(), "-", ""),
		}
		// if countrycode and phone number both are numbers, set WAId = countryCode+phoneNumber
		if helper.IsDigitsOnly(countryCode) && helper.IsDigitsOnly(phoneNumber) {
			customer.WAId = countryCode + phoneNumber
		}
		// for sg only
		if len(contact.PhoneNumber) == 8 && (strings.HasPrefix(contact.PhoneNumber, "8") || strings.HasPrefix(contact.PhoneNumber, "9")) {
			customer.WAId = "65" + contact.PhoneNumber
			customer.CountryCode = "65"
			customer.PhoneNumber = contact.PhoneNumber
		}
		customers = append(customers, customer)
		result.ImportedCount++
	}
	if err := transaction.CustomerRepository().InsertBulk(ctx, customers); err != nil {
		return dto.NewFailedResponse[*ImportResult](http.StatusInternalServerError, types.ExceptionMessageInternalServerError)
	}
	if err := transaction.CommitTransaction(); err != nil {
		return dto.NewFailedResponse[*ImportResult](http.StatusInternalServerError, types.ExceptionMessageInternalServerError)
	}
	committed = true
	result.Message = fmt.Sprintf("%d contacts successfully imported, %d skipped. Some contacts may miss country code, please amend accordingly.", result.ImportedCount, len(result.Skipped))
	return dto.NewSuccessResponse(result)
}

func parsePhoneNumber(phoneNumber string) (string, string) {
	// if phone number has ( and ), the number inside () is the country code
	if strings.HasPrefix(phoneNumber, "(") {
		closingParenthesis := strings.Index(phoneNumber, ")")
		if closingParenthesis > 1 {
			countryCode := phoneNumber[1:closingParenthesis]
			nationalNumber := phoneNumber[closingParenthesis+1:]
			if helper.IsDigitsOnly(countryCode) && helper.IsDigitsOnly(nationalNumber) {
				return countryCode, nationalNumber
			}
		}
	}
	countryCode, nationalNumber, err := helper.GetCountryCodeAndPhoneNumberFromWAId(phoneNumber)
	if err != nil {
		return ".", phoneNumber
	}
	return countryCode, nationalNumber
}

func (contact Contact) Tags() []string {
	categories := strings.Split(contact.Categories, ",")
	tags := make([]string, 0, len(categories))
	for _, category := range categories {
		category = strings.TrimSpace(category)
		if category != "" {
			tags = append(tags, category)
		}
	}
	return tags
}

func (contact Contact) AdditionalData() map[string]any {
	data := map[string]any{}
	if contact.Email != "" {
		data["email"] = contact.Email
	}
	if contact.Address != "" {
		data["address"] = contact.Address
	}
	if contact.Organization != "" {
		data["organization"] = contact.Organization
	}
	if contact.JobTitle != "" {
		data["job_title"] = contact.JobTitle
	}
	if contact.Birthday != "" {
		data["birthday"] = contact.Birthday
	}
	if contact.Anniversary != "" {
		data["anniversary"] = contact.Anniversary
	}
	if contact.Gender != "" {
		data["gender"] = contact.Gender
	}
	if contact.TimeZone != "" {
		data["time_zone"] = contact.TimeZone
	}
	// if contact.Photo != "" {
	// 	data["photo"] = contact.Photo
	// }
	if contact.URL != "" {
		data["url"] = contact.URL
	}
	return data
}

func (Import) APISettings() feature.APISettings {
	return feature.NewAPISettings(
		"Import customers",
		"Imports contacts and stores profile details in additional data.",
		types.HttpRequestTypeJSON,
		http.MethodPost,
		"/v1/customers/import",
		true,
		true,
		types.APITagCustomer,
		[]feature.APIError{
			feature.NewAPIError(*exception.NewCustomException("you are not authenticated", http.StatusForbidden)),
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageInternalServerError, http.StatusInternalServerError)),
		},
	)
}
