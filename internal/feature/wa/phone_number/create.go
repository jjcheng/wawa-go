package feature_wa_phone_number

import (
	"context"
	"net/http"
	"strings"

	dao_wa "github.com/jjcheng/wawa-go/internal/dao/wa"
	"github.com/jjcheng/wawa-go/internal/dto"
	dto_wa "github.com/jjcheng/wawa-go/internal/dto/wa"
	"github.com/jjcheng/wawa-go/internal/exception"
	feature_wa "github.com/jjcheng/wawa-go/internal/feature/wa"
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
		errors = append(errors, exception.NewInputException("meta_business_portfolio_id", "missing meta business portfolio id"))
	}
	if create.MetaPhoneNumberId == "" {
		errors = append(errors, exception.NewInputException("meta_phone_number_id", "missing meta phone number id"))
	}
	if create.MetaWABAId == "" {
		errors = append(errors, exception.NewInputException("meta_waba_id", "missing meta WABA id"))
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
	whatsapp, tokenEx := feature_wa.ClientForWABA(ctx, dependencies, create.MetaWABAId)
	if tokenEx != nil {
		return dto.NewFailedResponse[*dto_wa.PhoneNumber](tokenEx.StatusCode, tokenEx.Message)
	}
	displayPhoneNumber, displayName, err := whatsapp.GetDisplayPhoneNumberAndName(ctx, create.MetaPhoneNumberId)
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
