package feature_wa_message

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/jjcheng/wawa-go/internal/dto"
	dto_account "github.com/jjcheng/wawa-go/internal/dto/account"
	"github.com/jjcheng/wawa-go/internal/exception"
	"github.com/jjcheng/wawa-go/internal/feature"
	"github.com/jjcheng/wawa-go/internal/service"
	"github.com/jjcheng/wawa-go/internal/types"
	"gorm.io/gorm"
)

type MessageType string

type RecipientType string

const (
	RecipientTypeIndividual RecipientType = "individual"
	RecipientTypeGroup      RecipientType = "group"

	MessageTypeText        MessageType = "text"
	MessageTypeImage       MessageType = "image"
	MessageTypeAudio       MessageType = "audio"
	MessageTypeVideo       MessageType = "video"
	MessageTypeDocument    MessageType = "document"
	MessageTypeSticker     MessageType = "sticker"
	MessageTypeLocation    MessageType = "location"
	MessageTypeContacts    MessageType = "contacts"
	MessageTypeInteractive MessageType = "interactive"
	MessageTypeTemplate    MessageType = "template"
	MessageTypeReaction    MessageType = "reaction"
)

type Create struct {
	MessagingProduct string           `json:"messaging_product" val:"required" description:"always whatsapp"`
	RecipientType    RecipientType    `json:"recipient_type" val:"required" description:"type of the recipient. individual or group"`
	To               string           `json:"to" val:"required" description:"recipient's phone number or wa_id" example:"6590000000"`
	PhoneNumberID    string           `json:"phone_number_id" description:"set dynamically based on current user, any value is ignored"`
	Type             MessageType      `json:"type" val:"required" description:"one of the enum types"`
	Context          *MessageContext  `json:"context,omitempty" description:"if it's replying a previous message"`
	Text             *TextObject      `json:"text,omitempty" description:"only present if type is text"`
	Image            *MediaObject     `json:"image,omitempty" description:"only present if type is image"`
	Audio            *MediaObject     `json:"audio,omitempty" description:"only present if type is audio"`
	Video            *MediaObject     `json:"video,omitempty" description:"only present if type is video"`
	Document         *MediaObject     `json:"document,omitempty" description:"only present if type is document"`
	Sticker          *MediaObject     `json:"sticker,omitempty" description:"only present if type is sticker"`
	Location         *LocationObject  `json:"location,omitempty" description:"only present if type is location"`
	Contacts         []map[string]any `json:"contacts,omitempty" description:"only present if type is contacts"`
	Interactive      *InteractiveBody `json:"interactive,omitempty" description:"only if the message require user action, set the rest parameters to nil"`
	Template         *TemplateObject  `json:"template,omitempty" description:"only if the message is from a template, set the rest parameters to nil"`
	Reaction         *ReactionObject  `json:"reaction,omitempty" description:"only if the message is an emoji reaction to a previous message, an empty string is used to remove your existing reaction from that message. Set the rest including context to nil"`
}

type MessageContext struct {
	MessageID string `json:"message_id"`
}

type TextObject struct {
	Body       string `json:"body"`
	PreviewURL bool   `json:"preview_url,omitempty"`
}

type MediaObject struct {
	ID       string `json:"id,omitempty"`
	Link     string `json:"link,omitempty"`
	Caption  string `json:"caption,omitempty"`
	Filename string `json:"filename,omitempty"`
}

type LocationObject struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Name      string  `json:"name,omitempty"`
	Address   string  `json:"address,omitempty"`
	URL       string  `json:"url,omitempty"`
}

type ReactionObject struct {
	MessageID string `json:"message_id"`
	Emoji     string `json:"emoji"`
}

type TemplateObject struct {
	Name       string              `json:"name"`
	Language   TemplateLanguage    `json:"language"`
	Components []TemplateComponent `json:"components,omitempty"`
}

type TemplateLanguage struct {
	Policy string `json:"policy,omitempty"`
	Code   string `json:"code"`
}

type TemplateComponent struct {
	Type       string           `json:"type"`
	SubType    string           `json:"sub_type,omitempty"`
	Index      string           `json:"index,omitempty"`
	Parameters []map[string]any `json:"parameters,omitempty"`
}

type InteractiveBody struct {
	Type   string             `json:"type"`
	Header *InteractiveHeader `json:"header,omitempty"`
	Body   *InteractiveText   `json:"body,omitempty"`
	Footer *InteractiveText   `json:"footer,omitempty"`
	Action *InteractiveAction `json:"action,omitempty"`
}

type InteractiveHeader struct {
	Type     string       `json:"type"`
	Text     string       `json:"text,omitempty"`
	Image    *MediaObject `json:"image,omitempty"`
	Video    *MediaObject `json:"video,omitempty"`
	Document *MediaObject `json:"document,omitempty"`
}

type InteractiveText struct {
	Text string `json:"text"`
}

type InteractiveAction struct {
	Button            string               `json:"button,omitempty"`
	Sections          []InteractiveSection `json:"sections,omitempty"`
	CatalogID         string               `json:"catalog_id,omitempty"`
	ProductRetailerID string               `json:"product_retailer_id,omitempty"`
	Name              string               `json:"name,omitempty"`
	Parameters        map[string]any       `json:"parameters,omitempty"`
}

type InteractiveSection struct {
	Title        string           `json:"title,omitempty"`
	Rows         []InteractiveRow `json:"rows,omitempty"`
	ProductItems []map[string]any `json:"product_items,omitempty"`
}

type InteractiveRow struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
}

func (create *Create) Validate() []exception.InputException {
	create.To = strings.TrimSpace(create.To)
	create.PhoneNumberID = strings.TrimSpace(create.PhoneNumberID)
	inputErrors := []exception.InputException{}
	if create.To == "" {
		inputErrors = append(inputErrors, exception.NewInputException("to", "missing recipient"))
	}
	if create.Type == "" {
		inputErrors = append(inputErrors, exception.NewInputException("type", "missing message type"))
	}
	return inputErrors
}

func (create Create) Handle(ctx context.Context, user *dto_account.User, dependencies *service.Dependencies) dto.Response[*service.WhatsAppMessageResponse] {
	if user == nil {
		return dto.NewFailedResponse[*service.WhatsAppMessageResponse](http.StatusForbidden, "you are not authenticated")
	}
	if inputErrors := create.Validate(); len(inputErrors) > 0 {
		return dto.NewInvalidInputResponse[*service.WhatsAppMessageResponse](inputErrors)
	}
	phoneNumbers, err := dependencies.UnitOfWork.WAUserPhoneNumberRepository().ListPhoneNumbersByUserId(ctx, user.Id)
	if err != nil {
		return dto.NewFailedResponse[*service.WhatsAppMessageResponse](http.StatusInternalServerError, types.ExceptionMessageInternalServerError)
	}
	if len(phoneNumbers) == 0 {
		return dto.NewFailedResponse[*service.WhatsAppMessageResponse](http.StatusUnauthorized, "you are not authorized to use a WhatsApp phone number")
	}
	create.PhoneNumberID = phoneNumbers[0].MetaPhoneNumberId
	businessPortfolio, err := dependencies.UnitOfWork.WAPhoneNumberRepository().GetBusinessPortfolioByMetaPhoneNumberId(ctx, create.PhoneNumberID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return dto.NewFailedResponse[*service.WhatsAppMessageResponse](http.StatusUnauthorized, "you are not authorized to use this WhatsApp phone number")
		}
		return dto.NewFailedResponse[*service.WhatsAppMessageResponse](http.StatusInternalServerError, types.ExceptionMessageInternalServerError)
	}
	create.MessagingProduct = "whatsapp"
	if create.RecipientType == "" {
		create.RecipientType = RecipientTypeIndividual
	}
	response, err := dependencies.Whatsapp.SendMessage(ctx, create.PhoneNumberID, create, businessPortfolio.AccessToken)
	if err != nil {
		return dto.NewFailedResponse[*service.WhatsAppMessageResponse](http.StatusBadGateway, types.ExceptionMessageBadGateway)
	}
	return dto.NewSuccessResponse(response)
}

func (Create) APISettings() feature.APISettings {
	return feature.NewAPISettings(
		"Send WhatsApp message",
		"Sends a WhatsApp message from a phone number available to the authenticated user.",
		types.HttpRequestTypeJSON,
		http.MethodPost,
		"/v1/wa/messages",
		true,
		true,
		types.APITagWA,
		[]feature.APIError{
			feature.NewAPIError(*exception.NewCustomException("you are not authenticated", http.StatusForbidden)),
			feature.NewAPIError(*exception.NewCustomException("phone number not found", http.StatusNotFound)),
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageBadGateway, http.StatusBadGateway)),
		},
	)
}
