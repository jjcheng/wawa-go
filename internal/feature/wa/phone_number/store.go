package feature_wa_phone_number

import (
	"context"
	"errors"
	"net/http"
	"strings"

	dao_wa "github.com/jjcheng/wawa-go/internal/dao/wa"
	"github.com/jjcheng/wawa-go/internal/dto"
	dto_account "github.com/jjcheng/wawa-go/internal/dto/account"
	dto_wa "github.com/jjcheng/wawa-go/internal/dto/wa"
	"github.com/jjcheng/wawa-go/internal/exception"
	"github.com/jjcheng/wawa-go/internal/helper"
	"github.com/jjcheng/wawa-go/internal/service"
	"github.com/jjcheng/wawa-go/internal/types"
	"gorm.io/gorm"
)

// Store only persists the phone number. Registering it on Meta is a non-transactional
// side effect and is performed by the caller after the database transaction commits.
type Store struct {
	MetaBusinessPortfolioId string `json:"meta_business_portfolio_id" val:"required" description:"returned in embeded signup"`
	MetaWABAId              string `json:"meta_waba_id" val:"required" description:"returned in embedded signup"`
	MetaPhoneNumberId       string `json:"meta_phone_number_id" val:"required" description:"returned in embeded signup"`
	PhoneNumber             string `json:"phone_number" val:"required" description:"display phone number returned by Meta"`
	Name                    string `json:"name" val:"required" description:"verified name returned by Meta"`
	UserId                  int32  `json:"user_id" val:"required" description:"user who is managing this phone number"`
}

func (store *Store) Validate() []exception.InputException {
	var errors []exception.InputException
	store.MetaBusinessPortfolioId = strings.TrimSpace(store.MetaBusinessPortfolioId)
	store.MetaPhoneNumberId = strings.TrimSpace(store.MetaPhoneNumberId)
	store.MetaWABAId = strings.TrimSpace(store.MetaWABAId)
	store.PhoneNumber = strings.TrimSpace(store.PhoneNumber)
	store.Name = strings.TrimSpace(store.Name)
	if store.MetaBusinessPortfolioId == "" {
		errors = append(errors, exception.NewInputException("meta_business_portfolio_id", "missing meta business portfolio id"))
	}
	if store.MetaPhoneNumberId == "" {
		errors = append(errors, exception.NewInputException("meta_phone_number_id", "missing meta phone number id"))
	}
	if store.MetaWABAId == "" {
		errors = append(errors, exception.NewInputException("meta_waba_id", "missing meta WABA id"))
	}
	if store.PhoneNumber == "" {
		errors = append(errors, exception.NewInputException("phone_number", "missing display phone number"))
	}
	if store.Name == "" {
		errors = append(errors, exception.NewInputException("name", "missing verified name"))
	}
	if store.UserId <= 0 {
		errors = append(errors, exception.NewInputException("user_id", "missing user id"))
	}
	return errors
}

func (store Store) Handle(ctx context.Context, _ *dto_account.User, dependencies *service.Dependencies) dto.Response[*dto_wa.PhoneNumber] {
	if errors := store.Validate(); len(errors) > 0 {
		return dto.NewInvalidInputResponse[*dto_wa.PhoneNumber](errors)
	}
	existing, err := dependencies.UnitOfWork.WAPhoneNumberRepository().GetByPhoneNumberId(ctx, store.MetaPhoneNumberId)
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return dto.NewFailedResponse[*dto_wa.PhoneNumber](http.StatusInternalServerError, types.ExceptionMessageInternalServerError)
		}
	}
	// update existing
	if existing != nil {
		if existing.MetaBusinessPortfolioId != store.MetaBusinessPortfolioId {
			return dto.NewFailedResponse[*dto_wa.PhoneNumber](http.StatusBadRequest, "existing phone number does not match the Meta business portfolio id")
		}
		if existing.MetaWABAId != store.MetaWABAId {
			return dto.NewFailedResponse[*dto_wa.PhoneNumber](http.StatusBadRequest, "existing phone number does not match the Meta WABA id")
		}
		if existing.UserId != store.UserId {
			return dto.NewFailedResponse[*dto_wa.PhoneNumber](http.StatusBadRequest, "existing phone number does not belong to the user")
		}
		existing.PhoneNumber = store.PhoneNumber
		existing.WAId = helper.NormalizeWAId(existing.PhoneNumber)
		existing.Name = store.Name
		if err := dependencies.UnitOfWork.WAPhoneNumberRepository().Update(ctx, existing); err != nil {
			return dto.NewFailedResponse[*dto_wa.PhoneNumber](http.StatusInternalServerError, types.ExceptionMessageInternalServerError)
		}
		d := dto_wa.NewPhoneNumber(*existing)
		return dto.NewSuccessResponse(&d)
	}
	phoneNumber := dao_wa.PhoneNumber{
		MetaBusinessPortfolioId: store.MetaBusinessPortfolioId,
		MetaWABAId:              store.MetaWABAId,
		MetaPhoneNumberId:       store.MetaPhoneNumberId,
		PhoneNumber:             store.PhoneNumber,
		Name:                    store.Name,
		UserId:                  store.UserId,
		WAId:                    helper.NormalizeWAId(store.PhoneNumber),
	}
	if err := dependencies.UnitOfWork.WAPhoneNumberRepository().Insert(ctx, &phoneNumber); err != nil {
		return dto.NewFailedResponse[*dto_wa.PhoneNumber](http.StatusInternalServerError, types.ExceptionMessageInternalServerError)
	}
	d := dto_wa.NewPhoneNumber(phoneNumber)
	return dto.NewSuccessResponse(&d)
}
