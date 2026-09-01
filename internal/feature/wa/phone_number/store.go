package feature_wa_phone_number

import (
	"context"
	"errors"
	"net/http"
	"strings"

	dao_wa "github.com/jjcheng/wawa-go/internal/dao/wa"
	"github.com/jjcheng/wawa-go/internal/dto"
	dto_wa "github.com/jjcheng/wawa-go/internal/dto/wa"
	"github.com/jjcheng/wawa-go/internal/exception"
	"github.com/jjcheng/wawa-go/internal/service"
	"github.com/jjcheng/wawa-go/internal/types"
	"gorm.io/gorm"
)

type Store struct {
	MetaBusinessPortfolioId string `json:"meta_business_portfolio_id" val:"required" description:"returned in embeded signup"`
	MetaWABAId              string `json:"meta_waba_id" val:"required" description:"returned in embedded signup"`
	MetaPhoneNumberId       string `json:"meta_phone_number_id" val:"required" description:"returned in embeded signup"`
}

func (store *Store) Validate() []exception.InputException {
	var errors []exception.InputException
	store.MetaBusinessPortfolioId = strings.TrimSpace(store.MetaBusinessPortfolioId)
	store.MetaPhoneNumberId = strings.TrimSpace(store.MetaPhoneNumberId)
	store.MetaWABAId = strings.TrimSpace(store.MetaWABAId)
	if store.MetaBusinessPortfolioId == "" {
		errors = append(errors, exception.NewInputException("meta_business_portfolio_id", "missing meta business portfolio id"))
	}
	if store.MetaPhoneNumberId == "" {
		errors = append(errors, exception.NewInputException("meta_phone_number_id", "missing meta phone number id"))
	}
	if store.MetaWABAId == "" {
		errors = append(errors, exception.NewInputException("meta_waba_id", "missing meta WABA id"))
	}
	return errors
}

func (store Store) Handle(ctx context.Context, _, dependencies *service.Dependencies) dto.Response[*dto_wa.PhoneNumber] {
	if errors := store.Validate(); len(errors) > 0 {
		return dto.NewInvalidInputResponse[*dto_wa.PhoneNumber](errors)
	}
	existing, err := dependencies.UnitOfWork.WAPhoneNumberRepository().GetByPhoneNumberId(ctx, store.MetaPhoneNumberId)
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return dto.NewFailedResponse[*dto_wa.PhoneNumber](http.StatusInternalServerError, types.ExceptionMessageInternalServerError)
		}
	}
	// get display phone number and name by whatsapp service
	businessPortfolio, err := dependencies.UnitOfWork.WABusinessPortfolioRepository().GetByMetaBusinessPortfolioId(ctx, store.MetaBusinessPortfolioId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return dto.NewFailedResponse[*dto_wa.PhoneNumber](http.StatusNotFound, "business portfolio not found")
		}
		return dto.NewFailedResponse[*dto_wa.PhoneNumber](http.StatusInternalServerError, types.ExceptionMessageInternalServerError)
	}
	displayPhoneNumber, displayName, err := dependencies.Whatsapp.GetDisplayPhoneNumberAndName(ctx, store.MetaPhoneNumberId, businessPortfolio.AccessToken)
	if err != nil {
		return dto.NewFailedResponse[*dto_wa.PhoneNumber](http.StatusBadGateway, types.ExceptionMessageBadGateway)
	}
	// update existing
	if existing != nil {
		if existing.MetaBusinessPortfolioId != store.MetaBusinessPortfolioId {
			return dto.NewFailedResponse[*dto_wa.PhoneNumber](http.StatusBadRequest, "existing phone number does not match the Meta business portfolio id")
		}
		if existing.MetaWABAId != store.MetaWABAId {
			return dto.NewFailedResponse[*dto_wa.PhoneNumber](http.StatusBadRequest, "existing phone number does not match the Meta WABA id")
		}
		existing.PhoneNumber = displayPhoneNumber
		existing.Name = displayName
		if err := dependencies.UnitOfWork.WAPhoneNumberRepository().Update(ctx, existing); err != nil {
			return dto.NewFailedResponse[*dto_wa.PhoneNumber](http.StatusInternalServerError, types.ExceptionMessageInternalServerError)
		}
		d := dto_wa.NewPhoneNumber(*existing)
		return dto.NewSuccessResponse(&d)
	} else {
		phoneNumber := dao_wa.PhoneNumber{
			MetaBusinessPortfolioId: store.MetaBusinessPortfolioId,
			MetaWABAId:              store.MetaWABAId,
			MetaPhoneNumberId:       store.MetaPhoneNumberId,
			PhoneNumber:             displayPhoneNumber,
			Name:                    displayName,
		}
		if err := dependencies.UnitOfWork.WAPhoneNumberRepository().Insert(ctx, &phoneNumber); err != nil {
			return dto.NewFailedResponse[*dto_wa.PhoneNumber](http.StatusInternalServerError, types.ExceptionMessageInternalServerError)
		}
		d := dto_wa.NewPhoneNumber(phoneNumber)
		return dto.NewSuccessResponse(&d)
	}
}
