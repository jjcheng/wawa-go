package feature_wa_phone_number

import (
	"context"
	"net/http"
	"strings"

	dao_wa "github.com/jjcheng/wawa-go/internal/dao/wa"
	"github.com/jjcheng/wawa-go/internal/dto"
	dto_wa "github.com/jjcheng/wawa-go/internal/dto/wa"
	"github.com/jjcheng/wawa-go/internal/exception"
	"github.com/jjcheng/wawa-go/internal/service"
)

type Create struct {
	MetaBusinessPortfolioId string `json:"meta_business_portfolio_id" val:"required" description:"returned in embeded signup"`
	MetaWABAId              string `json:"meta_waba_id" val:"required" description:"returned in embedded signup"`
	MetaPhoneNumberId       string `json:"meta_phone_number_id" val:"required" description:"returned in embeded signup"`
}

func (create *Create) Validate() []exception.InputException {
	var errors []exception.InputException
	create.MetaBusinessPortfolioId = strings.TrimSpace(create.MetaBusinessPortfolioId)
	create.MetaPhoneNumberId = strings.TrimSpace(create.MetaPhoneNumberId)
	create.MetaWABAId = strings.TrimSpace(create.MetaWABAId)
	if create.MetaBusinessPortfolioId == "" {
		errors = append(errors, exception.NewInputException("meta_business_portfolio_id", "missing meta_business_portfolio_id"))
	}
	if create.MetaPhoneNumberId == "" {
		errors = append(errors, exception.NewInputException("meta_phone_number_id", "missing meta_phone_number_id"))
	}
	if create.MetaWABAId == "" {
		errors = append(errors, exception.NewInputException("meta_waba_id", "missing meta_waba_id"))
	}
	return errors
}

func (create Create) Handle(ctx context.Context, _, dependencies *service.Dependencies) dto.Response[*dto_wa.PhoneNumber] {
	if errors := create.Validate(); len(errors) > 0 {
		return dto.NewInvalidInputResponse[*dto_wa.PhoneNumber](errors)
	}
	exists, ex := dependencies.UnitOfWork.WAPhoneNumberRepository().CheckExists(ctx, create.MetaPhoneNumberId)
	if ex != nil {
		return dto.NewFailedResponse[*dto_wa.PhoneNumber](ex.StatusCode, ex.Message)
	}
	if exists {
		return dto.NewFailedResponse[*dto_wa.PhoneNumber](http.StatusBadRequest, "phone number already exists")
	}
	// get display phone number and name by whatsapp service
	displayPhoneNumber, displayName, err := dependencies.Whatsapp.GetDisplayPhoneNumberAndName(ctx, create.MetaPhoneNumberId)
	if err != nil {
		return dto.NewFailedResponse[*dto_wa.PhoneNumber](http.StatusBadGateway, "error getting phone number details")
	}
	phoneNumber := dao_wa.PhoneNumber{
		MetaBusinessPortfolioId: create.MetaBusinessPortfolioId,
		MetaWABAId:              create.MetaWABAId,
		MetaPhoneNumberId:       create.MetaPhoneNumberId,
		PhoneNumber:             displayPhoneNumber,
		Name:                    displayName,
	}
	if err := dependencies.UnitOfWork.WAPhoneNumberRepository().Insert(ctx, &phoneNumber); err != nil {
		dependencies.Logger.ErrorFunction(err, create)
		return dto.NewFailedResponse[*dto_wa.PhoneNumber](http.StatusInternalServerError, "error creating phone number")
	}
	d := dto_wa.NewPhoneNumber(phoneNumber)
	return dto.NewSuccessResponse(&d)
}
