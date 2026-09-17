package feature_wa_message

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	dao_wa "github.com/jjcheng/wawa-go/internal/dao/wa"
	"github.com/jjcheng/wawa-go/internal/dto"
	dto_account "github.com/jjcheng/wawa-go/internal/dto/account"
	dto_wa "github.com/jjcheng/wawa-go/internal/dto/wa"
	"github.com/jjcheng/wawa-go/internal/exception"
	"github.com/jjcheng/wawa-go/internal/feature"
	"github.com/jjcheng/wawa-go/internal/helper"
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
	CustomerId          int32            `json:"customer_id" val:"required" description:"id of the customer"`
	Type                MessageType      `json:"type" val:"required" description:"one of the enum types"`
	Context             *MessageContext  `json:"context,omitempty" description:"if it's replying a previous message"`
	Text                *TextObject      `json:"text,omitempty" description:"only present if type is text"`
	Image               *MediaObject     `json:"image,omitempty" description:"only present if type is image"`
	Audio               *MediaObject     `json:"audio,omitempty" description:"only present if type is audio"`
	Video               *MediaObject     `json:"video,omitempty" description:"only present if type is video"`
	Document            *MediaObject     `json:"document,omitempty" description:"only present if type is document"`
	Sticker             *MediaObject     `json:"sticker,omitempty" description:"only present if type is sticker"`
	Location            *LocationObject  `json:"location,omitempty" description:"only present if type is location"`
	Contacts            []map[string]any `json:"contacts,omitempty" description:"only present if type is contacts"`
	Interactive         *InteractiveBody `json:"interactive,omitempty" description:"only if the message require user action, set the rest parameters to nil"`
	Template            *map[string]any  `json:"template,omitempty" description:"only if the message is from a template, set the rest parameters to nil"`
	Reaction            *ReactionObject  `json:"reaction,omitempty" description:"only if the message is an emoji reaction to a previous message, an empty string is used to remove your existing reaction from that message. Set the rest including context to nil"`
	AttachmentURL       string           `json:"attachment_url,omitempty" description:"set message attachment_url"`
	CampaignRecipientId *int32           `json:"campaign_recipient_id,omitempty" description:"if it's from a campaign"`
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
	Type       string           `json:"type"`                 // header, body, or button
	SubType    string           `json:"sub_type,omitempty"`   // button only: url, quick_reply, copy_code, or voice_call
	Index      string           `json:"index,omitempty"`      // button only: zero-based position in the template, for example "0"
	Parameters []map[string]any `json:"parameters,omitempty"` // header/body: text, image, video, or document; button: text (url), payload (quick_reply), or coupon_code (copy_code)
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
	// origin and vcard are not part of Meta’s WhatsApp Cloud API contacts[] schema.
	for _, contact := range create.Contacts {
		delete(contact, "origin")
		delete(contact, "vcard")
	}
	inputErrors := []exception.InputException{}
	if create.CustomerId <= 0 {
		inputErrors = append(inputErrors, exception.NewInputException("customer_id", "missing customer_id"))
	}
	if create.Type == "" {
		inputErrors = append(inputErrors, exception.NewInputException("type", "missing message type"))
	}
	switch create.Type {
	case MessageTypeText:
		if create.Text == nil {
			inputErrors = append(inputErrors, exception.NewInputException("text", "text is missing for text message"))
		}
	case MessageTypeImage:
		if create.Image == nil {
			inputErrors = append(inputErrors, exception.NewInputException("image", "image is missing for image message"))
		}
	case MessageTypeAudio:
		if create.Audio == nil {
			inputErrors = append(inputErrors, exception.NewInputException("audio", "audio is missing for audio message"))
		}
	case MessageTypeVideo:
		if create.Video == nil {
			inputErrors = append(inputErrors, exception.NewInputException("video", "video is missing for video message"))
		}
	case MessageTypeDocument:
		if create.Document == nil {
			inputErrors = append(inputErrors, exception.NewInputException("document", "document is missing for document message"))
		}
	case MessageTypeSticker:
		if create.Sticker == nil {
			inputErrors = append(inputErrors, exception.NewInputException("sticker", "sticker is missing for sticker message"))
		}
	case MessageTypeLocation:
		if create.Location == nil {
			inputErrors = append(inputErrors, exception.NewInputException("location", "location is missing for location message"))
		}
	case MessageTypeContacts:
		if len(create.Contacts) == 0 {
			inputErrors = append(inputErrors, exception.NewInputException("contacts", "contacts are missing for contacts message"))
		}
	case MessageTypeInteractive:
		if create.Interactive == nil {
			inputErrors = append(inputErrors, exception.NewInputException("interactive", "interactive content is missing for interactive message"))
		}
	case MessageTypeTemplate:
		if create.Template == nil {
			inputErrors = append(inputErrors, exception.NewInputException("template", "template is missing for template message"))
		}
	case MessageTypeReaction:
		if create.Reaction == nil {
			inputErrors = append(inputErrors, exception.NewInputException("reaction", "reaction is missing for reaction message"))
		}
	case "":
	default:
		inputErrors = append(inputErrors, exception.NewInputException("type", "unsupported message type"))
	}
	return inputErrors
}

func (create Create) Handle(ctx context.Context, user *dto_account.User, dependencies *service.Dependencies) dto.Response[*dto_wa.Message] {
	if user == nil {
		return dto.NewFailedResponse[*dto_wa.Message](http.StatusForbidden, "you are not authenticated")
	}
	if inputErrors := create.Validate(); len(inputErrors) > 0 {
		return dto.NewInvalidInputResponse[*dto_wa.Message](inputErrors)
	}
	if user.WA == nil || user.WA.PhoneNumber_ == nil {
		return dto.NewFailedResponse[*dto_wa.Message](http.StatusUnauthorized, "you are not authorized to use a WhatsApp phone number")
	}
	if strings.TrimSpace(user.WA.BusinessPortfolioAccessToken) == "" {
		return dto.NewFailedResponse[*dto_wa.Message](http.StatusUnauthorized, "you are not authorized to use this WhatsApp phone number")
	}
	// get customer by customer id
	customer, err := dependencies.UnitOfWork.CustomerRepository().GetByIdAndUserId(ctx, create.CustomerId, user.Id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return dto.NewFailedResponse[*dto_wa.Message](http.StatusBadRequest, "customer not found")
		}
		return dto.NewFailedResponse[*dto_wa.Message](http.StatusInternalServerError, types.ExceptionMessageInternalServerError)
	}
	if customer.WAId == "" {
		return dto.NewFailedResponse[*dto_wa.Message](http.StatusBadRequest, "this customer has no WA ID, please edit the details")
	}
	payload, err := messagePayload(create)
	if err != nil {
		return dto.NewFailedResponse[*dto_wa.Message](http.StatusInternalServerError, types.ExceptionMessageInternalServerError)
	}
	payload["messaging_product"] = "whatsapp"
	payload["recipient_type"] = "individual"
	payload["to"] = customer.WAId
	delete(payload, "customer_id")
	token := strings.ReplaceAll(uuid.NewString(), "-", "")
	payload["biz_opaque_callback_data"] = token
	response, err := dependencies.Whatsapp.SendMessage(ctx, user.WA.PhoneNumber_.MetaPhoneNumberId, payload, user.WA.BusinessPortfolioAccessToken)
	// always insert message to db
	var sendError string
	messageStatus := types.WAMessageStatusAccepted
	if err != nil {
		sendError = err.Error()
		messageStatus = types.WAMessageStatusRejected
	} else if len(response.Messages) == 0 || strings.TrimSpace(response.Messages[0].ID) == "" {
		sendError = "WhatsApp did not return a message ID"
		messageStatus = types.WAMessageStatusRejected
	}
	timestamp := time.Now().Unix()
	message := dao_wa.Message{
		Sending:       true,
		PhoneNumberId: user.WA.PhoneNumber_.Id,
		CustomerId:    customer.Id,
		Timestamp:     timestamp,
		Type:          string(create.Type),
		Payload:       payload,
		Status:        messageStatus,
		AttachmentURL: create.AttachmentURL,
		Attempts:      1,
		ErrorMessage:  sendError,
		Token:         token,
	}
	// retry 3 min later if it's a campaign message and the error is a http request error (payload never go to Meta)
	// for any Meta returned error, no need to retry
	// set it to at least 3 mins becuase if the error was becuase meta did not return an id, it will return it in next 1-2 mins
	var requestHTTPError *helper.HTTPRequestError
	if sendError != "" && create.CampaignRecipientId != nil && errors.As(err, &requestHTTPError) {
		message.NextAttemptAt = helper.ConvertToPointer(time.Now().UTC().Add(3 * time.Minute))
	}
	if response != nil && len(response.Messages) > 0 && strings.TrimSpace(response.Messages[0].ID) != "" {
		message.WAMessageId = strings.TrimSpace(response.Messages[0].ID)
	}
	if err := dependencies.UnitOfWork.WAMessageRepository().Insert(ctx, &message); err != nil {
		return dto.NewFailedResponse[*dto_wa.Message](http.StatusInternalServerError, types.ExceptionMessageInternalServerError)
	}
	result := dto_wa.NewMessage(message)
	// publish the message even if it has error, so user is aware, this applies to campaign messages as well
	// if user is currently on the chat page
	dependencies.Ably.Publish("message", helper.GetChatChannelName(user.WA.PhoneNumber_.MetaPhoneNumberId, customer.Token), result)
	return dto.NewSuccessResponse(&result)
}

func messagePayload(message Create) (map[string]any, error) {
	data, err := json.Marshal(message)
	if err != nil {
		return nil, err
	}
	payload := map[string]any{}
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, err
	}
	return payload, nil
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
