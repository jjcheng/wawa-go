package feature_wa_message

import (
	"context"
	"net/http"
	"strings"

	"github.com/jjcheng/wawa-go/internal/dto"
	dto_account "github.com/jjcheng/wawa-go/internal/dto/account"
	"github.com/jjcheng/wawa-go/internal/exception"
	"github.com/jjcheng/wawa-go/internal/feature"
	"github.com/jjcheng/wawa-go/internal/service"
	"github.com/jjcheng/wawa-go/internal/types"
)

type Stream struct {
	PhoneNumberID       string `form:"phone_number_id" val:"required" description:"Meta ID of the business phone number"`
	CustomerPhoneNumber string `form:"customer_phone_number" val:"required" description:"Customer WhatsApp ID" example:"6590073708"`
}

type StreamSubscription struct {
	Events      <-chan service.WAMessageStreamEvent `json:"-"`
	Unsubscribe func()                              `json:"-"`
}

func (stream *Stream) Validate() []exception.InputException {
	stream.PhoneNumberID = strings.TrimSpace(stream.PhoneNumberID)
	stream.CustomerPhoneNumber = strings.TrimSpace(stream.CustomerPhoneNumber)
	inputErrors := []exception.InputException{}
	if stream.PhoneNumberID == "" {
		inputErrors = append(inputErrors, exception.NewInputException("phone_number_id", "missing phone number ID"))
	}
	if stream.CustomerPhoneNumber == "" {
		inputErrors = append(inputErrors, exception.NewInputException("customer_phone_number", "missing customer phone number"))
	}
	return inputErrors
}

func (stream Stream) Handle(ctx context.Context, user *dto_account.User, dependencies *service.Dependencies) dto.Response[*StreamSubscription] {
	if user == nil {
		return dto.NewFailedResponse[*StreamSubscription](http.StatusForbidden, "you are not authenticated")
	}
	if inputErrors := stream.Validate(); len(inputErrors) > 0 {
		return dto.NewInvalidInputResponse[*StreamSubscription](inputErrors)
	}
	phoneNumbers, err := dependencies.UnitOfWork.WAUserPhoneNumberRepository().ListPhoneNumbersByUserId(ctx, user.Id)
	if err != nil {
		return dto.NewFailedResponse[*StreamSubscription](http.StatusInternalServerError, types.ExceptionMessageInternalServerError)
	}
	for _, phoneNumber := range phoneNumbers {
		if phoneNumber.MetaPhoneNumberId == stream.PhoneNumberID {
			events, unsubscribe := dependencies.WAMessageStream.Subscribe(stream.PhoneNumberID, stream.CustomerPhoneNumber)
			return dto.NewSuccessResponse(&StreamSubscription{Events: events, Unsubscribe: unsubscribe})
		}
	}
	return dto.NewFailedResponse[*StreamSubscription](http.StatusUnauthorized, "you are not authorized to access this WhatsApp phone number")
}

func (Stream) APISettings() feature.APISettings {
	return feature.NewWebSocketAPISettings(
		"Stream WhatsApp conversation messages",
		"Establishes a WebSocket connection that emits message and status events for one customer conversation.",
		"/v1/wa/messages/stream",
		true,
		true,
		types.APITagWA,
		[]feature.APIError{
			feature.NewAPIError(*exception.NewCustomException("you are not authenticated", http.StatusForbidden)),
			feature.NewAPIError(*exception.NewCustomException("you are not authorized to access this WhatsApp phone number", http.StatusUnauthorized)),
		},
	)
}
