package service

import (
	"bytes"
	"context"
	cryptorand "crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"mime"
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

const (
	WhatsAppMessagingProduct = "whatsapp"

	WhatsAppMessageStatusRead = "read"

	WhatsAppPhoneNumberStatusConnected = "CONNECTED"
	WhatsAppPhoneNumberStatusPending   = "PENDING"
)

type WhatsAppMessageRequest struct {
	MessagingProduct string                   `json:"messaging_product"`
	Status           string                   `json:"status,omitempty"`
	MessageID        string                   `json:"message_id,omitempty"`
	TypingIndicator  *WhatsAppTypingIndicator `json:"typing_indicator,omitempty"`
}

type WhatsAppTypingIndicator struct {
	Type string `json:"type"`
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

type WhatsAppBusinessUseCaseUsage map[string][]WhatsAppBusinessUseCaseUsageEntry

type WhatsAppBusinessUseCaseUsageEntry struct {
	Type                        string `json:"type"`
	CallCount                   int    `json:"call_count"`
	TotalCPUTime                int    `json:"total_cputime"`
	TotalTime                   int    `json:"total_time"`
	EstimatedTimeToRegainAccess int    `json:"estimated_time_to_regain_access"`
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
	ErrorData      string `json:"error_data,omitempty"`
}

type WhatsAppAPIError struct {
	StatusCode int
	GraphError WhatsAppGraphAPIError
}

type WhatsAppRateLimitError struct {
	RetryAfterSeconds int
}

func (err *WhatsAppRateLimitError) Error() string {
	return fmt.Sprintf("WhatsApp rate limited; retry after %d seconds", err.RetryAfterSeconds)
}

func (err *WhatsAppAPIError) Error() string {
	if err.GraphError.ErrorUserMsg != "" {
		if err.GraphError.ErrorData != "" {
			return fmt.Sprintf("whatsapp api error %d (subcode %d): %s - %s (%s)", err.GraphError.Code, err.GraphError.ErrorSubcode, err.GraphError.Message, err.GraphError.ErrorUserMsg, err.GraphError.ErrorData)
		}
		return fmt.Sprintf("whatsapp api error %d (subcode %d): %s - %s", err.GraphError.Code, err.GraphError.ErrorSubcode, err.GraphError.Message, err.GraphError.ErrorUserMsg)
	}
	if err.GraphError.ErrorData != "" {
		return fmt.Sprintf("whatsapp api error %d (subcode %d): %s (%s)", err.GraphError.Code, err.GraphError.ErrorSubcode, err.GraphError.Message, err.GraphError.ErrorData)
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

type WhatsAppUploadSessionResponse struct {
	ID string `json:"id"`
}

type WhatsAppTemplateHeaderSampleUploadResponse struct {
	Handle string `json:"h"`
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

func (whatsapp *Whatsapp) DownloadFile(ctx context.Context, fileURL string, businessAccessToken string) ([]byte, string, string, error) {
	fileURL = strings.TrimSpace(fileURL)
	if fileURL == "" {
		return nil, "", "", fmt.Errorf("file URL is required")
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, fileURL, nil)
	if err != nil {
		return nil, "", "", fmt.Errorf("failed to create file download request: %w", err)
	}
	if businessAccessToken = strings.TrimSpace(businessAccessToken); businessAccessToken != "" {
		request.Header.Set("Authorization", "Bearer "+businessAccessToken)
	}
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return nil, "", "", fmt.Errorf("file download request failed: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return nil, "", "", fmt.Errorf("file download failed with status %d", response.StatusCode)
	}
	content, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, "", "", fmt.Errorf("failed to read file content: %w", err)
	}
	if len(content) == 0 {
		return nil, "", "", fmt.Errorf("file content is empty")
	}
	contentType := normalizeContentType(response.Header.Get("Content-Type"))
	filename := filenameFromURL(fileURL, contentType)
	if contentType == "" {
		contentType = mime.TypeByExtension(filepath.Ext(filename))
	}
	return content, contentType, filename, nil
}

func (whatsapp *Whatsapp) UploadTemplateHeaderSample(ctx context.Context, filename string, contentType string, content []byte, businessAccessToken string) (string, error) {
	filename = strings.TrimSpace(filename)
	contentType = strings.TrimSpace(contentType)
	if filename == "" {
		return "", fmt.Errorf("filename is required")
	}
	if contentType == "" {
		return "", fmt.Errorf("content type is required")
	}
	if len(content) == 0 {
		return "", fmt.Errorf("file content is required")
	}
	query := url.Values{}
	query.Set("file_name", filename)
	query.Set("file_length", strconv.Itoa(len(content)))
	query.Set("file_type", contentType)
	endpoint := fmt.Sprintf("%s/%s/%s/uploads?%s", whatsapp.baseURL, whatsapp.apiVersion, url.PathEscape(whatsapp.appID), query.Encode())
	var sessionResponse WhatsAppUploadSessionResponse
	if err := whatsapp.doJSONRequest(ctx, "create_template_header_sample_upload_session", http.MethodPost, endpoint, nil, &sessionResponse, businessAccessToken); err != nil {
		whatsapp.logger.ErrorFunction(err, filename, contentType)
		return "", err
	}
	if strings.TrimSpace(sessionResponse.ID) == "" {
		return "", fmt.Errorf("upload session ID is missing from Meta response")
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, fmt.Sprintf("%s/%s/%s", whatsapp.baseURL, whatsapp.apiVersion, url.PathEscape(sessionResponse.ID)), bytes.NewReader(content))
	if err != nil {
		return "", fmt.Errorf("failed to create template header sample upload request: %w", err)
	}
	request.Header.Set("Authorization", "OAuth "+strings.TrimSpace(businessAccessToken))
	request.Header.Set("Content-Type", contentType)
	request.Header.Set("file_offset", "0")
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return "", fmt.Errorf("template header sample upload request failed: %w", err)
	}
	defer response.Body.Close()
	responseBody, err := io.ReadAll(response.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read template header sample upload response: %w", err)
	}
	responseText := string(responseBody)
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return "", parseWhatsAppAPIError(response.StatusCode, &responseText)
	}
	var uploadResponse WhatsAppTemplateHeaderSampleUploadResponse
	if err := json.Unmarshal(responseBody, &uploadResponse); err != nil {
		return "", fmt.Errorf("failed to parse template header sample upload response: %w", err)
	}
	if strings.TrimSpace(uploadResponse.Handle) == "" {
		return "", fmt.Errorf("template header sample handle is missing from Meta response")
	}
	return uploadResponse.Handle, nil
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

func (whatsapp *Whatsapp) GetWABAUsage(ctx context.Context, wabaId string, start int64, end int64, granularity types.WAAnalyticsGranularity, businessAccessToken string) (*dto_wa.MessageAnalytics, error) {
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
	response.Analytics.TotalSent = helper.Sum(response.Analytics.DataPoints, func(dp dto_wa.MessageAnalyticsDataPoint) int {
		return int(dp.Sent)
	})
	response.Analytics.TotalDelivered = helper.Sum(response.Analytics.DataPoints, func(dp dto_wa.MessageAnalyticsDataPoint) int {
		return int(dp.Delivered)
	})
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

func (whatsapp *Whatsapp) GetPhoneNumberUsage(ctx context.Context, wabaId string, waIds []string, start int64, end int64, granularity types.WAAnalyticsGranularity, businessAccessToken string) (*dto_wa.MessageAnalytics, error) {
	if len(waIds) == 0 || len(waIds) > 10 {
		return nil, fmt.Errorf("waIds must contain between 1 and 10 phone numbers")
	}
	analytics, err := whatsapp.getPhoneNumerUsage(ctx, "get_phone_numbers_usage", wabaId, waIds, start, end, granularity, businessAccessToken)
	if err != nil {
		whatsapp.logger.ErrorFunction(err, wabaId, waIds)
		return nil, err
	}
	analytics.TotalSent = helper.Sum(analytics.DataPoints, func(d dto_wa.MessageAnalyticsDataPoint) int {
		return int(d.Sent)
	})
	analytics.TotalDelivered = helper.Sum(analytics.DataPoints, func(d dto_wa.MessageAnalyticsDataPoint) int {
		return int(d.Delivered)
	})
	return analytics, nil
}

func (whatsapp *Whatsapp) getPhoneNumerUsage(ctx context.Context, requestType string, wabaId string, waIds []string, start int64, end int64, granularity types.WAAnalyticsGranularity, businessAccessToken string) (*dto_wa.MessageAnalytics, error) {
	query := url.Values{}
	phoneNumberStrings := helper.Distinct(helper.Map(waIds, strconv.Quote))
	query.Set("fields", fmt.Sprintf("analytics.start(%d).end(%d).granularity(%s).phone_numbers([%s])", start, end, granularity, strings.Join(phoneNumberStrings, ",")))
	endpoint := fmt.Sprintf("%s/%s/%s?%s", whatsapp.baseURL, whatsapp.apiVersion, url.PathEscape(wabaId), query.Encode())
	var response struct {
		Analytics dto_wa.MessageAnalytics `json:"analytics"`
	}
	if err := whatsapp.doJSONRequest(ctx, requestType, http.MethodGet, endpoint, nil, &response, businessAccessToken); err != nil {
		return nil, err
	}
	return &response.Analytics, nil
}

func filenameFromURL(fileURL string, contentType string) string {
	filename := "template-header-sample"
	if parsedURL, err := url.Parse(fileURL); err == nil {
		if base := strings.TrimSpace(filepath.Base(parsedURL.Path)); base != "" && base != "." && base != "/" {
			filename = base
		}
	}
	if filepath.Ext(filename) == "" {
		if extensions, err := mime.ExtensionsByType(strings.TrimSpace(contentType)); err == nil && len(extensions) > 0 {
			filename += extensions[0]
		}
	}
	return filename
}

func normalizeContentType(contentType string) string {
	contentType = strings.TrimSpace(contentType)
	if contentType == "" {
		return ""
	}
	mediaType, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		return contentType
	}
	return mediaType
}

func (whatsapp *Whatsapp) ListTemplates(ctx context.Context, wabaId string, nameOrContent string, category types.WATemplateCategory, language string, status types.WATemplateStatus, qualityScore types.WATemplateQualityScore, before string, after string, limit int, businessAccessToken string) ([]dto_wa.Template, *dto_wa.TemplatePaging, error) {
	wabaId = strings.TrimSpace(wabaId)
	if wabaId == "" {
		return nil, nil, fmt.Errorf("wabaId is required")
	}
	query := url.Values{}
	query.Set("fields", "id,name,status,category,language,parameter_format,components,quality_score,rejected_reason,previous_category")
	if nameOrContent = strings.TrimSpace(nameOrContent); nameOrContent != "" {
		query.Set("name_or_content", nameOrContent)
	}
	if category != "" {
		query.Set("category", string(category))
	}
	if language = strings.TrimSpace(language); language != "" {
		query.Set("language", language)
	}
	if status != "" {
		query.Set("status", string(status))
	}
	if qualityScore != "" {
		query.Set("quality_score", string(qualityScore))
	}
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
	for i := range response.Data {
		response.Data[i].PreviewHTML = response.Data[i].HTML(true, false)
		response.Data[i].RawHTML = response.Data[i].HTML(false, false)
		response.Data[i].PreviewDarkHTML = response.Data[i].HTML(true, true)
		response.Data[i].RawDarkHTML = response.Data[i].HTML(false, true)
		response.Data[i].SendComponents = response.Data[i].GetSendComponents()
	}
	return response.Data, response.Paging, nil
}

func (whatsapp *Whatsapp) GetTemplate(ctx context.Context, templateID string, businessAccessToken string) (*dto_wa.Template, error) {
	templateID = strings.TrimSpace(templateID)
	if templateID == "" {
		return nil, fmt.Errorf("templateID is required")
	}
	query := url.Values{}
	query.Set("fields", "id,name,status,category,language,parameter_format,components,quality_score,rejected_reason,previous_category")
	endpoint := fmt.Sprintf("%s/%s/%s?%s", whatsapp.baseURL, whatsapp.apiVersion, url.PathEscape(templateID), query.Encode())
	var template dto_wa.Template
	if err := whatsapp.doJSONRequest(ctx, "get_template", http.MethodGet, endpoint, nil, &template, businessAccessToken); err != nil {
		whatsapp.logger.ErrorFunction(err, templateID)
		return nil, err
	}
	template.PreviewHTML = template.HTML(true, false)
	template.RawHTML = template.HTML(false, false)
	template.PreviewDarkHTML = template.HTML(true, true)
	template.RawDarkHTML = template.HTML(false, true)
	template.SendComponents = template.GetSendComponents()
	return &template, nil
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

func (whatsapp *Whatsapp) GetTemplateUsage(ctx context.Context, wabaId string, start string, end string, templateIds []string, granularity types.WAAnalyticsGranularity, businessAccessToken string) ([]dto_wa.TemplateAnalytics, error) {
	return whatsapp.getTemplateAnalytics(ctx, "get_template_usage", wabaId, start, end, templateIds, "sent,delivered,read,clicked", granularity, businessAccessToken)
}

func (whatsapp *Whatsapp) getTemplateAnalytics(ctx context.Context, requestType string, wabaId string, start string, end string, templateIds []string, metricTypes string, granularity types.WAAnalyticsGranularity, businessAccessToken string) ([]dto_wa.TemplateAnalytics, error) {
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
	query.Set("granularity", string(granularity))
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
		for index := range response.Data {
			response.Data[index].TotalSent = helper.Sum(response.Data[index].DataPoints, func(dataPoint dto_wa.TemplateAnalyticsDataPoint) int {
				return int(dataPoint.Sent)
			})
			response.Data[index].TotalDelivered = helper.Sum(response.Data[index].DataPoints, func(dataPoint dto_wa.TemplateAnalyticsDataPoint) int {
				return int(dataPoint.Delivered)
			})
			response.Data[index].TotalRead = helper.Sum(response.Data[index].DataPoints, func(dataPoint dto_wa.TemplateAnalyticsDataPoint) int {
				return int(dataPoint.Read)
			})
			response.Data[index].TotalClicked = helper.Sum(response.Data[index].DataPoints, func(dataPoint dto_wa.TemplateAnalyticsDataPoint) int {
				return int(dataPoint.Clicked)
			})
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

func (whatsapp *Whatsapp) SendMessage(ctx context.Context, phoneNumberID string, payload any, businessAccessToken string) (*WhatsAppMessageResponse, error) {
	var response WhatsAppMessageResponse
	err := whatsapp.doJSONRequest(ctx, "send_message", http.MethodPost, whatsapp.buildEndpoint(phoneNumberID, "messages"), payload, &response, businessAccessToken)
	if err != nil {
		whatsapp.logger.ErrorFunction(err, phoneNumberID)
		return nil, err
	}
	return &response, nil
}

func (whatsapp *Whatsapp) MarkAsRead(ctx context.Context, phoneNumberID string, messageID string, businessAccessToken string) (*WhatsAppMessageResponse, error) {
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
	statusCode, responseBody, responseHeader, err := helper.RequestHTTP(ctx, endpoint, method, &headers, bodyMap)
	if err != nil {
		return err
	}
	var usage *WhatsAppBusinessUseCaseUsage
	if responseHeader != nil {
		if businessUseCaseUsage := responseHeader.Get("X-Business-Use-Case-Usage"); businessUseCaseUsage != "" {
			if err := json.Unmarshal([]byte(businessUseCaseUsage), &usage); err != nil {
				whatsapp.logger.Warnf("failed to parse X-Business-Use-Case-Usage header: %v", err)
			}
		}
	}
	if responseBody != nil && strings.TrimSpace(*responseBody) != "" {
		if err := saveWhatsAppRawResponse(requestType, endpoint, method, bodyMap, *responseBody, usage); err != nil {
			whatsapp.logger.Warnf("failed to save WhatsApp raw response: %v", err)
		}
	}
	if statusCode < 200 || statusCode >= 300 {
		if statusCode == http.StatusTooManyRequests {
			if retryAfterSeconds := maxEstimatedTimeToRegainAccess(usage); retryAfterSeconds > 0 {
				return &WhatsAppRateLimitError{RetryAfterSeconds: retryAfterSeconds}
			}
		}
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

func maxEstimatedTimeToRegainAccess(usage *WhatsAppBusinessUseCaseUsage) int {
	if usage == nil {
		return 0
	}
	maxSeconds := 0
	for _, entries := range *usage {
		for _, entry := range entries {
			if entry.EstimatedTimeToRegainAccess > maxSeconds {
				maxSeconds = entry.EstimatedTimeToRegainAccess
			}
		}
	}
	return maxSeconds
}

func saveWhatsAppRawResponse(requestType string, endpoint string, method string, requestBody any, responseBody string, usage *WhatsAppBusinessUseCaseUsage) error {
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
	if usage != nil {
		exchange["usage"] = *usage
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
