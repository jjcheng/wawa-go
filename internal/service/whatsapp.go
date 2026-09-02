package service

import (
	"bytes"
	"context"
	cryptorand "crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/jjcheng/wawa-go/internal/cfg"
	dto_wa "github.com/jjcheng/wawa-go/internal/dto/wa"
	"github.com/jjcheng/wawa-go/internal/helper"
	"github.com/jjcheng/wawa-go/internal/types"
)

type Whatsapp struct {
	logger     *Logger
	baseURL    string
	apiVersion string
	appID      string
	appSecret  string
}

func NewWhatsapp(logger *Logger) *Whatsapp {
	config := cfg.Default().WhatsApp
	return &Whatsapp{
		logger:     logger,
		baseURL:    strings.TrimRight(strings.TrimSpace(config.BaseURL), "/"),
		apiVersion: strings.TrimSpace(config.APIVersion),
		appID:      strings.TrimSpace(config.AppID),
		appSecret:  strings.TrimSpace(config.AppSecret),
	}
}

type WhatsAppMessageType string

const (
	WhatsAppMessagingProduct = "whatsapp"

	WhatsAppRecipientIndividual = "individual"
	WhatsAppRecipientGroup      = "group"

	WhatsAppMessageTypeText        WhatsAppMessageType = "text"
	WhatsAppMessageTypeImage       WhatsAppMessageType = "image"
	WhatsAppMessageTypeAudio       WhatsAppMessageType = "audio"
	WhatsAppMessageTypeVideo       WhatsAppMessageType = "video"
	WhatsAppMessageTypeDocument    WhatsAppMessageType = "document"
	WhatsAppMessageTypeSticker     WhatsAppMessageType = "sticker"
	WhatsAppMessageTypeLocation    WhatsAppMessageType = "location"
	WhatsAppMessageTypeContacts    WhatsAppMessageType = "contacts"
	WhatsAppMessageTypeInteractive WhatsAppMessageType = "interactive"
	WhatsAppMessageTypeTemplate    WhatsAppMessageType = "template"
	WhatsAppMessageTypeReaction    WhatsAppMessageType = "reaction"

	WhatsAppMessageStatusRead = "read"

	WhatsAppPhoneNumberStatusConnected = "CONNECTED"
	WhatsAppPhoneNumberStatusPending   = "PENDING"
)

type WhatsAppMessageRequest struct {
	MessagingProduct string                   `json:"messaging_product"`
	RecipientType    string                   `json:"recipient_type,omitempty"`
	To               string                   `json:"to,omitempty"`
	PhoneNumberID    string                   `json:"phone_number_id,omitempty"`
	Type             WhatsAppMessageType      `json:"type,omitempty"`
	Context          *WhatsAppMessageContext  `json:"context,omitempty"`
	Text             *WhatsAppTextObject      `json:"text,omitempty"`
	Image            *WhatsAppMediaObject     `json:"image,omitempty"`
	Audio            *WhatsAppMediaObject     `json:"audio,omitempty"`
	Video            *WhatsAppMediaObject     `json:"video,omitempty"`
	Document         *WhatsAppMediaObject     `json:"document,omitempty"`
	Sticker          *WhatsAppMediaObject     `json:"sticker,omitempty"`
	Location         *WhatsAppLocationObject  `json:"location,omitempty"`
	Contacts         []map[string]any         `json:"contacts,omitempty"`
	Interactive      *WhatsAppInteractiveBody `json:"interactive,omitempty"`
	Template         *WhatsAppTemplateObject  `json:"template,omitempty"`
	Reaction         *WhatsAppReactionObject  `json:"reaction,omitempty"`

	Status          string                   `json:"status,omitempty"`
	MessageID       string                   `json:"message_id,omitempty"`
	TypingIndicator *WhatsAppTypingIndicator `json:"typing_indicator,omitempty"`
}

type WhatsAppMessageContext struct {
	MessageID string `json:"message_id"`
}

type WhatsAppTextObject struct {
	Body       string `json:"body"`
	PreviewURL bool   `json:"preview_url,omitempty"`
}

type WhatsAppMediaObject struct {
	ID       string `json:"id,omitempty"`
	Link     string `json:"link,omitempty"`
	Caption  string `json:"caption,omitempty"`
	Filename string `json:"filename,omitempty"`
}

type WhatsAppLocationObject struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Name      string  `json:"name,omitempty"`
	Address   string  `json:"address,omitempty"`
	URL       string  `json:"url,omitempty"`
}

type WhatsAppReactionObject struct {
	MessageID string `json:"message_id"`
	Emoji     string `json:"emoji"`
}

type WhatsAppTypingIndicator struct {
	Type string `json:"type"`
}

type WhatsAppTemplateObject struct {
	Name       string                      `json:"name"`
	Language   WhatsAppTemplateLanguage    `json:"language"`
	Components []WhatsAppTemplateComponent `json:"components,omitempty"`
}

type WhatsAppTemplateLanguage struct {
	Policy string `json:"policy,omitempty"`
	Code   string `json:"code"`
}

type WhatsAppTemplateComponent struct {
	Type       string           `json:"type"`
	SubType    string           `json:"sub_type,omitempty"`
	Index      string           `json:"index,omitempty"`
	Parameters []map[string]any `json:"parameters,omitempty"`
}

type WhatsAppInteractiveBody struct {
	Type   string                     `json:"type"`
	Header *WhatsAppInteractiveHeader `json:"header,omitempty"`
	Body   *WhatsAppInteractiveText   `json:"body,omitempty"`
	Footer *WhatsAppInteractiveText   `json:"footer,omitempty"`
	Action *WhatsAppInteractiveAction `json:"action,omitempty"`
}

type WhatsAppInteractiveHeader struct {
	Type     string               `json:"type"`
	Text     string               `json:"text,omitempty"`
	Image    *WhatsAppMediaObject `json:"image,omitempty"`
	Video    *WhatsAppMediaObject `json:"video,omitempty"`
	Document *WhatsAppMediaObject `json:"document,omitempty"`
}

type WhatsAppInteractiveText struct {
	Text string `json:"text"`
}

type WhatsAppInteractiveAction struct {
	Button            string                       `json:"button,omitempty"`
	Sections          []WhatsAppInteractiveSection `json:"sections,omitempty"`
	CatalogID         string                       `json:"catalog_id,omitempty"`
	ProductRetailerID string                       `json:"product_retailer_id,omitempty"`
	Name              string                       `json:"name,omitempty"`
	Parameters        map[string]any               `json:"parameters,omitempty"`
}

type WhatsAppInteractiveSection struct {
	Title        string                   `json:"title,omitempty"`
	Rows         []WhatsAppInteractiveRow `json:"rows,omitempty"`
	ProductItems []map[string]any         `json:"product_items,omitempty"`
}

type WhatsAppInteractiveRow struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
}

type WhatsAppMessageResponse struct {
	MessagingProduct string                    `json:"messaging_product"`
	Contacts         []WhatsAppContactResponse `json:"contacts"`
	Messages         []WhatsAppSentMessage     `json:"messages"`
}

type WhatsAppContactResponse struct {
	Input string `json:"input"`
	WaID  string `json:"wa_id"`
}

type WhatsAppSentMessage struct {
	ID            string `json:"id"`
	MessageStatus string `json:"message_status,omitempty"`
}

type WhatsAppMessageHistoryQuery struct {
	MessageID string
	Fields    string
	Limit     int
	After     string
	Before    string
}

type WhatsAppMessageHistoryResponse struct {
	Data   []WhatsAppMessageHistory `json:"data"`
	Paging *WhatsAppPaging          `json:"paging,omitempty"`
}

type WhatsAppMessageHistory struct {
	ID        string                    `json:"id"`
	MessageID string                    `json:"message_id"`
	Events    *WhatsAppMessageEventsSet `json:"events,omitempty"`
}

type WhatsAppMessageEventsSet struct {
	Data   []WhatsAppMessageEvent `json:"data"`
	Paging *WhatsAppPaging        `json:"paging,omitempty"`
}

type WhatsAppMessageEvent struct {
	ID                 string               `json:"id"`
	DeliveryStatus     string               `json:"delivery_status"`
	Timestamp          int64                `json:"timestamp"`
	WebhookUpdateState string               `json:"webhook_update_state,omitempty"`
	Application        *WhatsAppApplication `json:"application,omitempty"`
	WebhookURI         string               `json:"webhook_uri,omitempty"`
	ErrorDescription   string               `json:"error_description,omitempty"`
}

type WhatsAppMessageHistoryEventsResponse struct {
	Data   []WhatsAppMessageHistoryEventEdge `json:"data"`
	Paging *WhatsAppPaging                   `json:"paging,omitempty"`
}

type WhatsAppMessageHistoryEventEdge struct {
	Cursor string               `json:"cursor"`
	Node   WhatsAppHistoryEvent `json:"node"`
}

type WhatsAppHistoryEvent struct {
	ID                  string               `json:"id"`
	DeliveryStatus      string               `json:"delivery_status"`
	ErrorDescription    string               `json:"error_description,omitempty"`
	OccurrenceTimestamp int64                `json:"occurrence_timestamp"`
	StatusTimestamp     int64                `json:"status_timestamp,omitempty"`
	Application         *WhatsAppApplication `json:"application,omitempty"`
}

type WhatsAppApplication struct {
	ID   string `json:"id"`
	Name string `json:"name,omitempty"`
}

type WhatsAppPaging struct {
	Cursors  *WhatsAppPagingCursors `json:"cursors,omitempty"`
	Previous string                 `json:"previous,omitempty"`
	Next     string                 `json:"next,omitempty"`
}

type WhatsAppPagingCursors struct {
	Before string `json:"before,omitempty"`
	After  string `json:"after,omitempty"`
}

type WhatsAppGraphAPIErrorResponse struct {
	Error WhatsAppGraphAPIError `json:"error"`
}

type WhatsAppGraphAPIError struct {
	Message        string `json:"message"`
	Type           string `json:"type"`
	Code           int    `json:"code"`
	ErrorSubcode   int    `json:"error_subcode,omitempty"`
	FBTraceID      string `json:"fbtrace_id,omitempty"`
	IsTransient    bool   `json:"is_transient,omitempty"`
	ErrorUserTitle string `json:"error_user_title,omitempty"`
	ErrorUserMsg   string `json:"error_user_msg,omitempty"`
}

type WhatsAppAPIError struct {
	StatusCode int
	GraphError WhatsAppGraphAPIError
}

func (err *WhatsAppAPIError) Error() string {
	if err.GraphError.ErrorUserMsg != "" {
		return fmt.Sprintf("whatsapp api error %d (subcode %d): %s - %s", err.GraphError.Code, err.GraphError.ErrorSubcode, err.GraphError.Message, err.GraphError.ErrorUserMsg)
	}
	return fmt.Sprintf("whatsapp api error %d (subcode %d): %s", err.GraphError.Code, err.GraphError.ErrorSubcode, err.GraphError.Message)
}

type WhatsAppBusinessResponse struct {
	Name string `json:"name"`
}

type WhatsAppWABAResponse struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Currency   string `json:"currency"`
	TimezoneID string `json:"timezone_id"`
}

type WhatsAppWABAOwnerBusinessInfo struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type WhatsAppWABADetailsResponse struct {
	ID                  string                         `json:"id"`
	Name                string                         `json:"name"`
	Status              string                         `json:"status"`
	AccountReviewStatus string                         `json:"account_review_status"`
	OwnerBusinessInfo   *WhatsAppWABAOwnerBusinessInfo `json:"owner_business_info,omitempty"`
}

type WhatsAppPhoneNumberDetailsResponse struct {
	DisplayPhoneNumber     string `json:"display_phone_number"`
	VerifiedName           string `json:"verified_name"`
	ID                     string `json:"id"`
	Status                 string `json:"status"`
	CodeVerificationStatus string `json:"code_verification_status"`
}

type WhatsAppMediaResponse struct {
	ID       string `json:"id"`
	URL      string `json:"url"`
	MimeType string `json:"mime_type"`
	Sha256   string `json:"sha256"`
	FileSize int64  `json:"file_size"`
}

type WhatsAppMediaUploadResponse struct {
	ID string `json:"id"`
}

type WhatsAppBusinessTokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type,omitempty"`
}

func (whatsapp *Whatsapp) ExchangeAccessToken(ctx context.Context, authorizationCode string) (string, error) {
	payload := map[string]string{
		"client_id":     whatsapp.appID,
		"client_secret": whatsapp.appSecret,
		"code":          authorizationCode,
		"grant_type":    "authorization_code",
	}
	endpoint := fmt.Sprintf("%s/%s/oauth/access_token", whatsapp.baseURL, whatsapp.apiVersion)
	var tokenResponse WhatsAppBusinessTokenResponse
	if err := whatsapp.doJSONRequest(ctx, "exchange_access_token", http.MethodPost, endpoint, payload, &tokenResponse, ""); err != nil {
		whatsapp.logger.ErrorFunction(err)
		return "", err
	}
	if strings.TrimSpace(tokenResponse.AccessToken) == "" {
		return "", fmt.Errorf("business access token is missing from Meta response")
	}
	return tokenResponse.AccessToken, nil
}

func (whatsapp *Whatsapp) DownloadMedia(ctx context.Context, mediaID string, businessAccessToken string) ([]byte, string, error) {
	mediaID = strings.TrimSpace(mediaID)
	if mediaID == "" {
		return nil, "", fmt.Errorf("media ID is required")
	}
	metadataEndpoint := fmt.Sprintf("%s/%s/%s", whatsapp.baseURL, whatsapp.apiVersion, url.PathEscape(mediaID))
	var metadata WhatsAppMediaResponse
	if err := whatsapp.doJSONRequest(ctx, "download_media", http.MethodGet, metadataEndpoint, nil, &metadata, businessAccessToken); err != nil {
		whatsapp.logger.ErrorFunction(err, mediaID)
		return nil, "", err
	}
	if strings.TrimSpace(metadata.URL) == "" {
		return nil, "", fmt.Errorf("media URL is missing from Meta response")
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, metadata.URL, nil)
	if err != nil {
		return nil, "", fmt.Errorf("failed to create media download request: %w", err)
	}
	request.Header.Set("Authorization", "Bearer "+businessAccessToken)
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return nil, "", fmt.Errorf("media download request failed: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return nil, "", fmt.Errorf("media download failed with status %d", response.StatusCode)
	}
	content, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, "", fmt.Errorf("failed to read media content: %w", err)
	}
	contentType := response.Header.Get("Content-Type")
	if contentType == "" {
		contentType = metadata.MimeType
	}
	return content, contentType, nil
}

func (whatsapp *Whatsapp) UploadMedia(ctx context.Context, phoneNumberID string, filename string, contentType string, content []byte, businessAccessToken string) (string, error) {
	phoneNumberID = strings.TrimSpace(phoneNumberID)
	filename = strings.TrimSpace(filename)
	contentType = strings.TrimSpace(contentType)
	if phoneNumberID == "" {
		return "", fmt.Errorf("phone number ID is required")
	}
	if filename == "" {
		return "", fmt.Errorf("filename is required")
	}
	if contentType == "" {
		return "", fmt.Errorf("content type is required")
	}
	if len(content) == 0 {
		return "", fmt.Errorf("media content is required")
	}
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	if err := writer.WriteField("messaging_product", WhatsAppMessagingProduct); err != nil {
		return "", fmt.Errorf("failed to write messaging product: %w", err)
	}
	partHeaders := make(textproto.MIMEHeader)
	partHeaders.Set("Content-Disposition", fmt.Sprintf(`form-data; name="file"; filename="%s"`, filepath.Base(filename)))
	partHeaders.Set("Content-Type", contentType)
	part, err := writer.CreatePart(partHeaders)
	if err != nil {
		return "", fmt.Errorf("failed to create media form file: %w", err)
	}
	if _, err := part.Write(content); err != nil {
		return "", fmt.Errorf("failed to write media content: %w", err)
	}
	if err := writer.Close(); err != nil {
		return "", fmt.Errorf("failed to close media form: %w", err)
	}
	endpoint := fmt.Sprintf("%s/%s/%s/media", whatsapp.baseURL, whatsapp.apiVersion, url.PathEscape(phoneNumberID))
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, body)
	if err != nil {
		return "", fmt.Errorf("failed to create media upload request: %w", err)
	}
	request.Header.Set("Authorization", "Bearer "+businessAccessToken)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return "", fmt.Errorf("media upload request failed: %w", err)
	}
	defer response.Body.Close()
	responseBody, err := io.ReadAll(response.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read media upload response: %w", err)
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		responseText := string(responseBody)
		return "", parseWhatsAppAPIError(response.StatusCode, &responseText)
	}
	var uploadResponse WhatsAppMediaUploadResponse
	if err := json.Unmarshal(responseBody, &uploadResponse); err != nil {
		return "", fmt.Errorf("failed to parse media upload response: %w", err)
	}
	if strings.TrimSpace(uploadResponse.ID) == "" {
		return "", fmt.Errorf("media ID is missing from Meta response")
	}
	return uploadResponse.ID, nil
}

// RegisterPhoneNumber registers the number on Cloud API and returns the generated two-step verification PIN,
// which must be persisted because the same PIN is required to re-register the number later.
func (whatsapp *Whatsapp) RegisterPhoneNumber(ctx context.Context, phoneNumberID string, businessAccessToken string) (string, error) {
	phoneNumberID = strings.TrimSpace(phoneNumberID)
	if phoneNumberID == "" {
		return "", fmt.Errorf("phone number ID is required")
	}
	pinNumber, err := cryptorand.Int(cryptorand.Reader, big.NewInt(1000000))
	if err != nil {
		return "", fmt.Errorf("failed to generate registration PIN: %w", err)
	}
	pin := fmt.Sprintf("%06d", pinNumber.Int64())
	payload := map[string]any{
		"messaging_product": WhatsAppMessagingProduct,
		"pin":               pin,
	}
	endpoint := whatsapp.buildEndpoint(phoneNumberID, "register")
	if err := whatsapp.doJSONRequest(ctx, "register_phone_number", http.MethodPost, endpoint, payload, nil, businessAccessToken); err != nil {
		whatsapp.logger.ErrorFunction(err, phoneNumberID)
		return "", err
	}
	return pin, nil
}

// SubscribeApp subscribes this app to the WABA's webhooks; without it no inbound message or status callbacks are delivered.
func (whatsapp *Whatsapp) SubscribeApp(ctx context.Context, wabaId string, businessAccessToken string) error {
	wabaId = strings.TrimSpace(wabaId)
	if wabaId == "" {
		return fmt.Errorf("wabaId is required")
	}
	endpoint := whatsapp.buildEndpoint(wabaId, "subscribed_apps")
	if err := whatsapp.doJSONRequest(ctx, "subscribe_app", http.MethodPost, endpoint, nil, nil, businessAccessToken); err != nil {
		whatsapp.logger.ErrorFunction(err, wabaId)
		return err
	}
	return nil
}

func (whatsapp *Whatsapp) RemovePhoneNumber(ctx context.Context, phoneNumberID string, businessAccessToken string) error {
	phoneNumberID = strings.TrimSpace(phoneNumberID)
	if phoneNumberID == "" {
		return fmt.Errorf("phone number ID is required")
	}
	endpoint := fmt.Sprintf("%s/%s/%s", whatsapp.baseURL, whatsapp.apiVersion, url.PathEscape(phoneNumberID))
	if err := whatsapp.doJSONRequest(ctx, "remove_phone_number", http.MethodDelete, endpoint, nil, nil, businessAccessToken); err != nil {
		whatsapp.logger.ErrorFunction(err, phoneNumberID)
		return err
	}
	return nil
}

func (whatsapp *Whatsapp) GetBusinessName(ctx context.Context, metaBusinessPortfolioId string, businessAccessToken string) (string, error) {
	metaBusinessPortfolioId = strings.TrimSpace(metaBusinessPortfolioId)
	if metaBusinessPortfolioId == "" {
		return "", fmt.Errorf("metaBusinessPortfolioId is required")
	}
	endpoint := fmt.Sprintf("%s/%s/%s", whatsapp.baseURL, whatsapp.apiVersion, url.PathEscape(metaBusinessPortfolioId))
	query := url.Values{}
	query.Set("fields", "name")
	endpoint += "?" + query.Encode()
	var response WhatsAppBusinessResponse
	if err := whatsapp.doJSONRequest(ctx, "get_business_name", http.MethodGet, endpoint, nil, &response, businessAccessToken); err != nil {
		whatsapp.logger.ErrorFunction(err, metaBusinessPortfolioId)
		return "", err
	}
	if strings.TrimSpace(response.Name) == "" {
		return "", fmt.Errorf("business name is missing from the WhatsApp API response")
	}
	return response.Name, nil
}

// GetWABA reads the WABA together with its owning business portfolio. Unlike the business portfolio
// edges this only needs whatsapp_business_management, not business_management.
func (whatsapp *Whatsapp) GetWABA(ctx context.Context, wabaId string, businessAccessToken string) (*WhatsAppWABADetailsResponse, error) {
	wabaId = strings.TrimSpace(wabaId)
	if wabaId == "" {
		return nil, fmt.Errorf("wabaId is required")
	}
	query := url.Values{}
	query.Set("fields", "id,name,status,account_review_status,owner_business_info")
	endpoint := fmt.Sprintf("%s/%s/%s?%s", whatsapp.baseURL, whatsapp.apiVersion, url.PathEscape(wabaId), query.Encode())
	var response WhatsAppWABADetailsResponse
	if err := whatsapp.doJSONRequest(ctx, "get_waba", http.MethodGet, endpoint, nil, &response, businessAccessToken); err != nil {
		whatsapp.logger.ErrorFunction(err, wabaId)
		return nil, err
	}
	if response.OwnerBusinessInfo == nil || strings.TrimSpace(response.OwnerBusinessInfo.ID) == "" {
		return nil, fmt.Errorf("owner business info is missing from the WhatsApp API response")
	}
	return &response, nil
}

func (whatsapp *Whatsapp) GetWABAUsage(ctx context.Context, wabaId string, start int64, end int64, granularity string, businessAccessToken string) (*dto_wa.MessageAnalytics, error) {
	wabaId = strings.TrimSpace(wabaId)
	if wabaId == "" {
		return nil, fmt.Errorf("wabaId is required")
	}
	query := url.Values{}
	query.Set("fields", fmt.Sprintf("analytics.start(%d).end(%d).granularity(%s)", start, end, granularity))
	endpoint := fmt.Sprintf("%s/%s/%s?%s", whatsapp.baseURL, whatsapp.apiVersion, url.PathEscape(wabaId), query.Encode())
	var response struct {
		Analytics dto_wa.MessageAnalytics `json:"analytics"`
	}
	if err := whatsapp.doJSONRequest(ctx, "get_waba_usage", http.MethodGet, endpoint, nil, &response, businessAccessToken); err != nil {
		whatsapp.logger.ErrorFunction(err, wabaId)
		return nil, err
	}
	return &response.Analytics, nil
}

func (whatsapp *Whatsapp) GetAllPhoneNumbersByWABAId(ctx context.Context, wabaId string, businessAccessToken string) ([]WhatsAppPhoneNumberDetailsResponse, error) {
	query := url.Values{}
	query.Set("fields", "id,display_phone_number,verified_name")
	endpoint := fmt.Sprintf("%s/%s/%s/phone_numbers?%s", whatsapp.baseURL, whatsapp.apiVersion, url.PathEscape(wabaId), query.Encode())
	phoneNumbers := make([]WhatsAppPhoneNumberDetailsResponse, 0)
	for endpoint != "" {
		var response struct {
			Data   []WhatsAppPhoneNumberDetailsResponse `json:"data"`
			Paging *dto_wa.AnalyticsPaging              `json:"paging,omitempty"`
		}
		if err := whatsapp.doJSONRequest(ctx, "get_all_phone_numbers_by_waba_id", http.MethodGet, endpoint, nil, &response, businessAccessToken); err != nil {
			whatsapp.logger.ErrorFunction(err, wabaId)
			return nil, err
		}
		phoneNumbers = append(phoneNumbers, response.Data...)
		if response.Paging == nil {
			break
		}
		endpoint = strings.TrimSpace(response.Paging.Next)
	}
	return phoneNumbers, nil
}

func (whatsapp *Whatsapp) GetPhoneNumberUsage(ctx context.Context, wabaId string, start int64, end int64, granularity string, businessAccessToken string) ([]dto_wa.PhoneNumberMessageAnalytics, error) {
	wabaId = strings.TrimSpace(wabaId)
	metaPhoneNumbers, err := whatsapp.GetAllPhoneNumbersByWABAId(ctx, wabaId, businessAccessToken)
	if err != nil {
		return nil, err
	}
	phoneNumbers := make([]dto_wa.PhoneNumberMessageAnalytics, 0, len(metaPhoneNumbers))
	for _, phoneNumber := range metaPhoneNumbers {
		displayPhoneNumber := strings.TrimSpace(phoneNumber.DisplayPhoneNumber)
		normalizedPhoneNumber := normalizeWhatsAppPhoneNumber(displayPhoneNumber)
		if normalizedPhoneNumber == "" {
			continue
		}
		analytics, err := whatsapp.getMessageAnalyticsForPhoneNumber(ctx, "get_phone_numbers_usage", wabaId, normalizedPhoneNumber, start, end, granularity, businessAccessToken)
		if err != nil {
			whatsapp.logger.ErrorFunction(err, wabaId, displayPhoneNumber)
			return nil, err
		}
		phoneNumbers = append(phoneNumbers, dto_wa.PhoneNumberMessageAnalytics{
			ID:                 phoneNumber.ID,
			DisplayPhoneNumber: displayPhoneNumber,
			VerifiedName:       phoneNumber.VerifiedName,
			Analytics:          *analytics,
		})
	}
	return phoneNumbers, nil
}

func normalizeWhatsAppPhoneNumber(phoneNumber string) string {
	return strings.Map(func(character rune) rune {
		if character >= '0' && character <= '9' {
			return character
		}
		return -1
	}, phoneNumber)
}

func (whatsapp *Whatsapp) getMessageAnalyticsForPhoneNumber(ctx context.Context, requestType string, wabaId string, phoneNumber string, start int64, end int64, granularity string, businessAccessToken string) (*dto_wa.MessageAnalytics, error) {
	query := url.Values{}
	query.Set("fields", fmt.Sprintf("analytics.start(%d).end(%d).granularity(%s).phone_numbers([%s])", start, end, granularity, strconv.Quote(phoneNumber)))
	endpoint := fmt.Sprintf("%s/%s/%s?%s", whatsapp.baseURL, whatsapp.apiVersion, url.PathEscape(wabaId), query.Encode())
	var response struct {
		Analytics dto_wa.MessageAnalytics `json:"analytics"`
	}
	if err := whatsapp.doJSONRequest(ctx, requestType, http.MethodGet, endpoint, nil, &response, businessAccessToken); err != nil {
		return nil, err
	}
	return &response.Analytics, nil
}

func (whatsapp *Whatsapp) ListTemplates(ctx context.Context, wabaId string, businessAccessToken string) ([]dto_wa.Template, error) {
	endpoint := fmt.Sprintf("%s/%s/%s/message_templates", whatsapp.baseURL, whatsapp.apiVersion, url.PathEscape(wabaId))
	query := url.Values{}
	query.Set("fields", "id,name,status,category,language,parameter_format,components,quality_score,rejected_reason,previous_category")
	endpoint += "?" + query.Encode()
	templates := []dto_wa.Template{}
	for endpoint != "" {
		var response dto_wa.TemplateListResponse
		if err := whatsapp.doJSONRequest(ctx, "list_templates", http.MethodGet, endpoint, nil, &response, businessAccessToken); err != nil {
			whatsapp.logger.ErrorFunction(err, wabaId)
			return nil, err
		}
		templates = append(templates, response.Data...)
		if response.Paging == nil {
			break
		}
		endpoint = strings.TrimSpace(response.Paging.Next)
	}
	return templates, nil
}

func (whatsapp *Whatsapp) ListTemplatesPage(ctx context.Context, wabaId string, before string, after string, limit int, businessAccessToken string) ([]dto_wa.Template, *dto_wa.TemplatePaging, error) {
	wabaId = strings.TrimSpace(wabaId)
	if wabaId == "" {
		return nil, nil, fmt.Errorf("wabaId is required")
	}
	query := url.Values{}
	query.Set("fields", "id,name,status,category,language,parameter_format,components,quality_score,rejected_reason,previous_category")
	if before = strings.TrimSpace(before); before != "" {
		query.Set("before", before)
	} else if after = strings.TrimSpace(after); after != "" {
		query.Set("after", after)
	}
	if limit > 0 {
		query.Set("limit", strconv.Itoa(limit))
	}
	endpoint := fmt.Sprintf("%s/%s/%s/message_templates?%s", whatsapp.baseURL, whatsapp.apiVersion, url.PathEscape(wabaId), query.Encode())
	var response dto_wa.TemplateListResponse
	if err := whatsapp.doJSONRequest(ctx, "list_templates_page", http.MethodGet, endpoint, nil, &response, businessAccessToken); err != nil {
		whatsapp.logger.ErrorFunction(err, wabaId)
		return nil, nil, err
	}
	return response.Data, response.Paging, nil
}

func (whatsapp *Whatsapp) EnableTemplateInsights(ctx context.Context, wabaId string, businessAccessToken string) error {
	wabaId = strings.TrimSpace(wabaId)
	if wabaId == "" {
		return fmt.Errorf("wabaId is required")
	}
	query := url.Values{}
	query.Set("is_enabled_for_insights", "true")
	endpoint := fmt.Sprintf("%s/%s/%s?%s", whatsapp.baseURL, whatsapp.apiVersion, url.PathEscape(wabaId), query.Encode())
	if err := whatsapp.doJSONRequest(ctx, "enable_template_insights", http.MethodPost, endpoint, nil, nil, businessAccessToken); err != nil {
		whatsapp.logger.ErrorFunction(err, wabaId)
		return err
	}
	return nil
}

func (whatsapp *Whatsapp) GetTemplateUsage(ctx context.Context, wabaId string, start string, end string, templateIds []string, businessAccessToken string) ([]dto_wa.TemplateAnalytics, error) {
	return whatsapp.getTemplateAnalytics(ctx, "get_template_usage", wabaId, start, end, templateIds, "sent,delivered,read,clicked", businessAccessToken)
}

func (whatsapp *Whatsapp) getTemplateAnalytics(ctx context.Context, requestType string, wabaId string, start string, end string, templateIds []string, metricTypes string, businessAccessToken string) ([]dto_wa.TemplateAnalytics, error) {
	wabaId = strings.TrimSpace(wabaId)
	if wabaId == "" {
		return nil, fmt.Errorf("wabaId is required")
	}
	if start == "" || end == "" {
		return nil, fmt.Errorf("start and end are required")
	}
	if len(templateIds) == 0 || len(templateIds) > 10 {
		return nil, fmt.Errorf("templateIds must contain between 1 and 10 template IDs")
	}
	query := url.Values{}
	query.Set("start", start)
	query.Set("end", end)
	query.Set("granularity", "daily")
	query.Set("metric_types", metricTypes)
	query.Set("use_waba_timezone", "true")
	query.Set("template_ids", "["+strings.Join(templateIds, ",")+"]")
	endpoint := fmt.Sprintf("%s/%s/%s/template_analytics?%s", whatsapp.baseURL, whatsapp.apiVersion, url.PathEscape(wabaId), query.Encode())
	analytics := make([]dto_wa.TemplateAnalytics, 0)
	insightsEnabled := false
	for endpoint != "" {
		var response dto_wa.TemplateAnalyticsListResponse
		err := whatsapp.doJSONRequest(ctx, requestType, http.MethodGet, endpoint, nil, &response, businessAccessToken)
		var apiErr *WhatsAppAPIError
		if err != nil && !insightsEnabled && errors.As(err, &apiErr) && apiErr.GraphError.ErrorSubcode == 4182004 {
			if enableErr := whatsapp.EnableTemplateInsights(ctx, wabaId, businessAccessToken); enableErr != nil {
				whatsapp.logger.ErrorFunction(enableErr, wabaId)
				return nil, enableErr
			}
			insightsEnabled = true
			err = whatsapp.doJSONRequest(ctx, requestType, http.MethodGet, endpoint, nil, &response, businessAccessToken)
		}
		if err != nil {
			whatsapp.logger.ErrorFunction(err, wabaId, templateIds)
			return nil, err
		}
		analytics = append(analytics, response.Data...)
		if response.Paging == nil {
			break
		}
		endpoint = strings.TrimSpace(response.Paging.Next)
	}
	return analytics, nil
}

func (whatsapp *Whatsapp) CreateTemplate(ctx context.Context, wabaId string, payload map[string]any, businessAccessToken string) (*dto_wa.Template, error) {
	endpoint := fmt.Sprintf("%s/%s/%s/message_templates", whatsapp.baseURL, whatsapp.apiVersion, url.PathEscape(wabaId))
	var response dto_wa.Template
	if err := whatsapp.doJSONRequest(ctx, "create_template", http.MethodPost, endpoint, payload, &response, businessAccessToken); err != nil {
		whatsapp.logger.ErrorFunction(err, wabaId, payload)
		return nil, err
	}
	response.MetaWABAId = wabaId
	return &response, nil
}

func (whatsapp *Whatsapp) DeleteTemplate(ctx context.Context, wabaId string, name string, templateId string, businessAccessToken string) error {
	query := url.Values{}
	query.Set("name", name)
	query.Set("hsm_id", templateId)
	endpoint := fmt.Sprintf("%s/%s/%s/message_templates?%s", whatsapp.baseURL, whatsapp.apiVersion, url.PathEscape(wabaId), query.Encode())
	if err := whatsapp.doJSONRequest(ctx, "delete_template", http.MethodDelete, endpoint, nil, nil, businessAccessToken); err != nil {
		whatsapp.logger.ErrorFunction(err, wabaId, name, templateId)
		return err
	}
	return nil
}

// GetPhoneNumberStatus returns the connection status and code verification status, e.g. CONNECTED and VERIFIED once registration succeeded.
func (whatsapp *Whatsapp) GetPhoneNumber(ctx context.Context, metaPhoneNumberId string, businessAccessToken string) (*WhatsAppPhoneNumberDetailsResponse, error) {
	endpoint := fmt.Sprintf("%s/%s/%s", whatsapp.baseURL, whatsapp.apiVersion, url.PathEscape(metaPhoneNumberId))
	query := url.Values{}
	query.Set("fields", "display_phone_number,verified_name,status,code_verification_status")
	endpoint += "?" + query.Encode()
	var response WhatsAppPhoneNumberDetailsResponse
	if err := whatsapp.doJSONRequest(ctx, "get_phone_number", http.MethodGet, endpoint, nil, &response, businessAccessToken); err != nil {
		whatsapp.logger.ErrorFunction(err, metaPhoneNumberId)
		return nil, err
	}
	if strings.TrimSpace(response.DisplayPhoneNumber) == "" || strings.TrimSpace(response.VerifiedName) == "" {
		return nil, fmt.Errorf("display phone number or verified name is missing from the WhatsApp API response")
	}
	return &response, nil
}

func (whatsapp *Whatsapp) ReceiveMessage(messageQueueService *MessageQueue, message *MessageQueueMessage, handler func(incomingMessage dto_wa.IncomingMessage) error, historyHandler func(wabaID string, incomingMessage dto_wa.IncomingMessage) error) {
	whatsapp.logger.Infof("SMQ listener received message: message_id=%s", message.MessageID)
	body := []byte(strings.TrimSpace(message.Body))
	if len(body) == 0 {
		whatsapp.logger.Warnf("SMQ message body is empty: message_id=%s", message.MessageID)
		return
	}
	if decodedBody, err := base64.StdEncoding.DecodeString(message.Body); err == nil && json.Valid(decodedBody) {
		body = decodedBody
	} else if err != nil {
		whatsapp.logger.Debugf("SMQ message body is not base64, treating as raw webhook body: message_id=%s", message.MessageID)
		return
	}
	incoming, err := helper.DeserializeJSON[dto_wa.Incoming](string(body))
	if err != nil {
		whatsapp.logger.Warnf("SMQ processor failed to decode whatsapp send-status payload: message_id=%s err=%v", message.MessageID, err)
		return
	}
	for _, entry := range incoming.Entry {
		for _, change := range entry.Changes {
			phoneNumberID := strings.TrimSpace(change.Value.Metadata.PhoneNumberID)
			if change.Field == "history" {
				for _, historyMessage := range change.Value.Messages {
					historyMessage.PhoneNumberID = phoneNumberID
					if err := historyHandler(entry.ID, historyMessage); err != nil {
						whatsapp.logger.Warnf("failed to store WhatsApp history message: message_id=%s err=%v", message.MessageID, err)
					}
				}
				continue
			}
			if len(change.Value.Messages) > 0 {
				for _, incomingMessage := range change.Value.Messages {
					incomingMessage.PhoneNumberID = phoneNumberID
					whatsapp.logger.Infof(
						"SMQ incoming whatsapp message: message_id=%s entry_id=%s field=%s from=%s type=%s wa_message_id=%s body=%s",
						message.MessageID,
						entry.ID,
						change.Field,
						incomingMessage.From,
						incomingMessage.Type,
						incomingMessage.ID,
						incomingMessage.Text.Body,
					)
					if incomingMessage.From == "" {
						continue
					}
					err := handler(incomingMessage)
					if err != nil {
						whatsapp.logger.Warnf("SMQ processor failed to handle incoming whatsapp message: message_id=%s from=%s err=%v", message.MessageID, incomingMessage.From, err)
					}
				}
			} else if len(change.Value.Statuses) > 0 {
				for _, status := range change.Value.Statuses {
					whatsapp.logger.Infof("wa status change: %s - %s", status.ID, status.Status)
				}
			}
		}
	}
	if err := messageQueueService.DeleteMessage(message.ReceiptHandle); err != nil {
		whatsapp.logger.Warnf("SMQ processor failed to delete message: message_id=%s err=%v", message.MessageID, err)
	}
}

func (whatsapp *Whatsapp) SendMessage(ctx context.Context, request *WhatsAppMessageRequest, businessAccessToken string) (*WhatsAppMessageResponse, error) {
	if strings.TrimSpace(request.To) == "" {
		return nil, fmt.Errorf("to is required")
	}
	if request.Type == "" {
		return nil, fmt.Errorf("type is required")
	}
	phoneNumberID := strings.TrimSpace(request.PhoneNumberID)
	request.MessagingProduct = WhatsAppMessagingProduct
	if strings.TrimSpace(request.RecipientType) == "" {
		request.RecipientType = WhatsAppRecipientIndividual
	}
	var response WhatsAppMessageResponse
	err := whatsapp.doJSONRequest(ctx, "send_message", http.MethodPost, whatsapp.buildEndpoint(phoneNumberID, "messages"), request, &response, businessAccessToken)
	if err != nil {
		whatsapp.logger.ErrorFunction(err, request)
		return nil, err
	}
	return &response, nil
}

func (whatsapp *Whatsapp) MarkAsRead(ctx context.Context, phoneNumberID string, messageID string, businessAccessToken string) (*WhatsAppMessageResponse, error) {
	messageID = strings.TrimSpace(messageID)
	request := &WhatsAppMessageRequest{
		MessagingProduct: WhatsAppMessagingProduct,
		Status:           WhatsAppMessageStatusRead,
		MessageID:        messageID,
	}
	var response WhatsAppMessageResponse
	err := whatsapp.doJSONRequest(ctx, "mark_as_read", http.MethodPost, whatsapp.buildEndpoint(phoneNumberID, "messages"), request, &response, businessAccessToken)
	if err != nil {
		whatsapp.logger.ErrorFunction(err, request)
		return nil, err
	}
	return &response, nil
}

func (whatsapp *Whatsapp) StartTyping(ctx context.Context, phoneNumberID string, messageID string, businessAccessToken string) (*WhatsAppMessageResponse, error) {
	request := &WhatsAppMessageRequest{
		MessagingProduct: WhatsAppMessagingProduct,
		Status:           WhatsAppMessageStatusRead,
		MessageID:        messageID,
		TypingIndicator: &WhatsAppTypingIndicator{
			Type: "text",
		},
	}
	var response WhatsAppMessageResponse
	err := whatsapp.doJSONRequest(ctx, "start_typeing", http.MethodPost, whatsapp.buildEndpoint(phoneNumberID, "messages"), request, &response, businessAccessToken)
	if err != nil {
		whatsapp.logger.ErrorFunction(err, request)
		return nil, err
	}
	return &response, nil
}

// shared
func (whatsapp *Whatsapp) buildEndpoint(phoneNumberID string, edge string) string {
	url := fmt.Sprintf("%s/%s/%s/%s", whatsapp.baseURL, whatsapp.apiVersion, phoneNumberID, strings.TrimPrefix(edge, "/"))
	return url
}

func (whatsapp *Whatsapp) doJSONRequest(ctx context.Context, requestType string, method string, endpoint string, payload any, target any, businessAccessToken string) error {
	businessAccessToken = strings.TrimSpace(businessAccessToken)
	headers := map[string]string{}
	if businessAccessToken != "" {
		headers["Authorization"] = "Bearer " + businessAccessToken
	}
	if userAgent := helper.GetUserAgent(ctx); userAgent != nil && strings.TrimSpace(*userAgent) != "" {
		headers["User-Agent"] = *userAgent
	}
	var bodyMap *map[string]any
	if payload != nil {
		data, err := json.Marshal(payload)
		if err != nil {
			return fmt.Errorf("failed to marshal request payload: %w", err)
		}
		requestBody := map[string]any{}
		if err := json.Unmarshal(data, &requestBody); err != nil {
			return fmt.Errorf("failed to convert request payload: %w", err)
		}
		bodyMap = &requestBody
	}
	statusCode, responseBody, _, err := helper.RequestHTTP(ctx, endpoint, method, &headers, bodyMap)
	if err != nil {
		return err
	}
	if responseBody != nil && strings.TrimSpace(*responseBody) != "" {
		if err := saveWhatsAppRawResponse(requestType, endpoint, method, bodyMap, *responseBody); err != nil {
			whatsapp.logger.Warnf("failed to save WhatsApp raw response: %v", err)
		}
	}
	if statusCode < 200 || statusCode >= 300 {
		return parseWhatsAppAPIError(statusCode, responseBody)
	}
	if target == nil || responseBody == nil || strings.TrimSpace(*responseBody) == "" {
		return nil
	}
	if err := json.Unmarshal([]byte(*responseBody), target); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}
	return nil
}

func saveWhatsAppRawResponse(requestType string, endpoint string, method string, requestBody any, responseBody string) error {
	if cfg.Default().Site.Environment != types.EnvironmentDevelop {
		return nil
	}
	if err := os.MkdirAll("files/wa", 0755); err != nil {
		return fmt.Errorf("failed to create raw response directory: %w", err)
	}
	filename := fmt.Sprintf("%s.json", requestType)
	var responseObject any
	if err := json.Unmarshal([]byte(responseBody), &responseObject); err != nil {
		return fmt.Errorf("failed to parse raw response: %w", err)
	}
	exchange := map[string]any{
		"type": requestType,
		"request": map[string]any{
			"method": method,
			"url":    endpoint,
			"body":   requestBody,
		},
		"response": responseObject,
	}
	data, err := json.MarshalIndent(exchange, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal raw exchange: %w", err)
	}
	return helper.WriteToFile(string(data), filepath.Join("files/wa", filename))
}

func parseWhatsAppAPIError(statusCode int, responseBody *string) error {
	if responseBody == nil || strings.TrimSpace(*responseBody) == "" {
		return fmt.Errorf("whatsapp api request failed with status %d", statusCode)
	}
	parsed, err := helper.DeserializeJSON[WhatsAppGraphAPIErrorResponse](*responseBody)
	if err != nil || parsed == nil || parsed.Error.Message == "" {
		return fmt.Errorf("whatsapp api request failed with status %d: %s", statusCode, strings.TrimSpace(*responseBody))
	}
	return &WhatsAppAPIError{StatusCode: statusCode, GraphError: parsed.Error}
}
