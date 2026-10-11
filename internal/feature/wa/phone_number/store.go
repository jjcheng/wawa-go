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

// only used after embedded signup
// Store only persists the phone number. Registering it on Meta is a non-transactional
// side effect and is performed by the caller after the database transaction commits.
type Store struct {
	BusinessAccountId   int32  `json:"-" val:"required" description:"id of the business account"`
	MetaPhoneNumberId   string `json:"-" val:"required" description:"returned in embeded signup"`
	DisplayPhoneNumber  string `json:"-" val:"required" description:"display phone number returned by Meta"`
	Name                string `json:"-" val:"required" description:"verified name returned by Meta"`
	IsInTransaction     bool   `json:"-" val:"required" description:"if unitOfWork has a transaction already"`
	BusinessAccessToken string `json:"-" val:"required" description:"exchanged in embedded signup"`
}

func (store *Store) Validate() []exception.InputException {
	var errors []exception.InputException
	store.MetaPhoneNumberId = strings.TrimSpace(store.MetaPhoneNumberId)
	store.DisplayPhoneNumber = strings.TrimSpace(store.DisplayPhoneNumber)
	store.Name = strings.TrimSpace(store.Name)
	if store.BusinessAccountId <= 0 {
		errors = append(errors, exception.NewInputException("business_account_id", "missing business account id"))
	}
	if store.MetaPhoneNumberId == "" {
		errors = append(errors, exception.NewInputException("meta_phone_number_id", "missing meta phone number id"))
	}
	if store.DisplayPhoneNumber == "" {
		errors = append(errors, exception.NewInputException("display_phone_number", "missing display phone number"))
	}
	if store.Name == "" {
		errors = append(errors, exception.NewInputException("name", "missing verified name"))
	}
	if store.BusinessAccessToken == "" {
		errors = append(errors, exception.NewInputException("business_access_token", "missing business access token"))
	}
	return errors
}

func (store Store) Handle(ctx context.Context, user *dto_account.User, dependencies *service.Dependencies) dto.Response[*dto_wa.PhoneNumber] {
	if user != nil && user.Type != types.UserTypeMaster {
		return dto.NewFailedResponse[*dto_wa.PhoneNumber](http.StatusUnauthorized, types.ExceptionMessageUnauthorized, nil)
	}
	if user != nil && user.BusinessAccountId != store.BusinessAccountId {
		return dto.NewFailedResponse[*dto_wa.PhoneNumber](http.StatusUnauthorized, types.ExceptionMessageUnauthorized, nil)
	}
	if errors := store.Validate(); len(errors) > 0 {
		return dto.NewInvalidInputResponse[*dto_wa.PhoneNumber](errors)
	}
	existing, err := dependencies.UnitOfWork.WAPhoneNumberRepository().GetByMetaPhoneNumberId(ctx, store.MetaPhoneNumberId)
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return dto.NewFailedResponse[*dto_wa.PhoneNumber](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
		}
	}
	// start syncing app state
	syncAppContactsRequestId, err := dependencies.Whatsapp.StartSyncingAppContacts(ctx, store.MetaPhoneNumberId, store.BusinessAccessToken)
	if err != nil {
		dependencies.Logger.ErrorFunction(err, store.MetaPhoneNumberId)
	}
	syncAppHistoryRequestId, err := dependencies.Whatsapp.StartSyncingAppHistory(ctx, store.MetaPhoneNumberId, store.BusinessAccessToken)
	if err != nil {
		dependencies.Logger.ErrorFunction(err, store.MetaPhoneNumberId)
	}
	// update existing
	if existing != nil {
		if existing.BusinessAccountId != store.BusinessAccountId {
			return dto.NewFailedResponse[*dto_wa.PhoneNumber](http.StatusConflict, "existing phone number has different business account id", nil)
		}
		existing.DisplayPhoneNumber = store.DisplayPhoneNumber
		existing.WAId = helper.NormalizeWAId(existing.DisplayPhoneNumber)
		existing.Name = store.Name
		existing.Status = types.WAPhoneNumberStatusConnected
		existing.ContactSyncRequestId = syncAppContactsRequestId
		existing.HistorySyncRequestId = syncAppHistoryRequestId
		if err := dependencies.UnitOfWork.WAPhoneNumberRepository().Update(ctx, existing); err != nil {
			return dto.NewFailedResponse[*dto_wa.PhoneNumber](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
		}
		d := dto_wa.NewPhoneNumber(*existing)
		return dto.NewSuccessResponse(&d)
	}
	// insert new
	phoneNumber := dao_wa.PhoneNumber{
		BusinessAccountId:    store.BusinessAccountId,
		MetaPhoneNumberId:    store.MetaPhoneNumberId,
		DisplayPhoneNumber:   store.DisplayPhoneNumber,
		Status:               types.WAPhoneNumberStatusConnected,
		Name:                 store.Name,
		WAId:                 helper.NormalizeWAId(store.DisplayPhoneNumber),
		ContactSyncRequestId: syncAppContactsRequestId,
		HistorySyncRequestId: syncAppHistoryRequestId,
	}
	if err := dependencies.UnitOfWork.WAPhoneNumberRepository().Insert(ctx, &phoneNumber); err != nil {
		return dto.NewFailedResponse[*dto_wa.PhoneNumber](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
	}
	d := dto_wa.NewPhoneNumber(phoneNumber)
	d.New = true // this is a new number
	return dto.NewSuccessResponse(&d)
}
