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
	BusinessAccountId  int32  `json:"business_account_id" val:"required" description:"id of the business account"`
	MetaPhoneNumberId  string `json:"meta_phone_number_id" val:"required" description:"returned in embeded signup"`
	DisplayPhoneNumber string `json:"display_phone_number" val:"required" description:"display phone number returned by Meta"`
	Name               string `json:"name" val:"required" description:"verified name returned by Meta"`
	IsInTransaction    bool   `json:"is_in_transaction" val:"required" description:"if unitOfWork has a transaction already"`
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
	// update existing
	if existing != nil {
		if existing.BusinessAccountId != store.BusinessAccountId {
			return dto.NewFailedResponse[*dto_wa.PhoneNumber](http.StatusConflict, "existing phone number has different business account id", nil)
		}
		existing.DisplayPhoneNumber = store.DisplayPhoneNumber
		existing.WAId = helper.NormalizeWAId(existing.DisplayPhoneNumber)
		existing.Name = store.Name
		if err := dependencies.UnitOfWork.WAPhoneNumberRepository().Update(ctx, existing); err != nil {
			return dto.NewFailedResponse[*dto_wa.PhoneNumber](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
		}
		// // assign to users
		// // get existing assignments
		// existingAssignments, err := transaction.AccountUserPhoneNumberRepository().ListByPhoneNumberId(ctx, existing.Id)
		// if err != nil {
		// 	return dto.NewFailedResponse[*dto_wa.PhoneNumber](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
		// }
		// for _, userId := range store.UserIds {
		// 	if existingAssignmentIndex := helper.IndexOf(existingAssignments, func(assignment dao_account.UserPhoneNumber) bool {
		// 		return assignment.UserId == userId
		// 	}); existingAssignmentIndex != nil {
		// 		// already assigned, skip
		// 		existingAssignments[*existingAssignmentIndex].Processed = true
		// 		continue
		// 	}
		// 	// insert new
		// 	newUserPhoneNumber := dao_account.UserPhoneNumber{
		// 		UserId:        userId,
		// 		PhoneNumberId: existing.Id,
		// 		Processed:     true,
		// 	}
		// 	if err := transaction.AccountUserPhoneNumberRepository().Insert(ctx, &newUserPhoneNumber); err != nil {
		// 		return dto.NewFailedResponse[*dto_wa.PhoneNumber](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
		// 	}
		// }
		// for i := range existingAssignments {
		// 	if existingAssignments[i].Processed {
		// 		continue
		// 	}
		// 	if err := transaction.AccountUserPhoneNumberRepository().DeleteById(ctx, existingAssignments[i].Id); err != nil {
		// 		return dto.NewFailedResponse[*dto_wa.PhoneNumber](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
		// 	}
		// }
		d := dto_wa.NewPhoneNumber(*existing)
		return dto.NewSuccessResponse(&d)
	}
	// insert new
	phoneNumber := dao_wa.PhoneNumber{
		BusinessAccountId:  store.BusinessAccountId,
		MetaPhoneNumberId:  store.MetaPhoneNumberId,
		DisplayPhoneNumber: store.DisplayPhoneNumber,
		Name:               store.Name,
		WAId:               helper.NormalizeWAId(store.DisplayPhoneNumber),
	}
	if err := dependencies.UnitOfWork.WAPhoneNumberRepository().Insert(ctx, &phoneNumber); err != nil {
		return dto.NewFailedResponse[*dto_wa.PhoneNumber](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
	}
	d := dto_wa.NewPhoneNumber(phoneNumber)
	d.New = true // this is a new number
	return dto.NewSuccessResponse(&d)
}
