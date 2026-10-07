package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/jjcheng/wawa-go/internal/cfg"
	"github.com/jjcheng/wawa-go/internal/exception"
	"github.com/jjcheng/wawa-go/internal/helper"
	"github.com/jjcheng/wawa-go/internal/types"
)

type Facebook struct {
	logger  *Logger
	baseURL string
}

func NewFacebook(logger *Logger) *Facebook {
	return &Facebook{
		logger:  logger,
		baseURL: "https://api.facebook.com",
	}
}

type AppAgentEligibilityResponse struct {
	IsEligible bool `json:"is_eligible"`
}

type AgentOnboardResponse struct {
	AgentId string `json:"agent_id"`
}

type AgentOffboardResponse struct {
	DeleteAgentId string `json:"deleted_agent_id"`
}

type AgentBugetsResponse struct {
	Budgets []AgentBudget `json:"budgets"`
}

type AgentBudget struct {
	ID         string `json:"budget_id"`
	Max        int    `json:"max_budget"`
	UnitType   string `json:"unit_type"`   // One of "token", "ai_turn"
	TimeWindow string `json:"time_window"` // One of "one_day", "seven_days", "fourteen_days", "thirty_days"
}

func (agentBudget *AgentBudget) Validate() []exception.InputException {
	var errors []exception.InputException
	if agentBudget.Max <= 0 {
		errors = append(errors, exception.NewInputException("max_budget", "missing max budget"))
	}
	if agentBudget.UnitType != "token" && agentBudget.UnitType != "ai_turn" {
		errors = append(errors, exception.NewInputException("unit_type", "invalid unit type, must be one of token or ai_turn"))
	}
	if agentBudget.TimeWindow != "one_day" && agentBudget.TimeWindow != "seven_days" && agentBudget.TimeWindow != "fourteen_days" && agentBudget.TimeWindow != "thirty_days" {
		errors = append(errors, exception.NewInputException("time_window", "invalid time window, must be one of one_day, seven_days, fourteen_days, thirty_days"))
	}
	return errors
}

type AgentPhoneNumberBusinessInfo struct {
	PaymentMethod       string                              `json:"payment_method"`
	ReturnPolicy        string                              `json:"return_policy"`
	PurchaseInfo        string                              `json:"purchase_info"`
	DeliveryAndShipping string                              `json:"delivery_and_shipping"`
	BusinessDescription string                              `json:"business_description"`
	ContactInfo         AgentPhoneNumberBusinessContactInfo `json:"contact_info"`
}

type AgentPhoneNumberBusinessContactInfo struct {
	Email          string `json:"email"`
	OperatingHours string `json:"hours_of_operation"`
	Address        string `json:"address"`
}

func (agentPhoneNumberBusinessInfo *AgentPhoneNumberBusinessInfo) Validate() []exception.InputException {
	var errors []exception.InputException
	agentPhoneNumberBusinessInfo.PaymentMethod = strings.TrimSpace(agentPhoneNumberBusinessInfo.PaymentMethod)
	agentPhoneNumberBusinessInfo.ReturnPolicy = strings.TrimSpace(agentPhoneNumberBusinessInfo.ReturnPolicy)
	agentPhoneNumberBusinessInfo.PurchaseInfo = strings.TrimSpace(agentPhoneNumberBusinessInfo.PurchaseInfo)
	agentPhoneNumberBusinessInfo.DeliveryAndShipping = strings.TrimSpace(agentPhoneNumberBusinessInfo.DeliveryAndShipping)
	agentPhoneNumberBusinessInfo.BusinessDescription = strings.TrimSpace(agentPhoneNumberBusinessInfo.BusinessDescription)
	agentPhoneNumberBusinessInfo.ContactInfo.Email = strings.TrimSpace(agentPhoneNumberBusinessInfo.ContactInfo.Email)
	agentPhoneNumberBusinessInfo.ContactInfo.OperatingHours = strings.TrimSpace(agentPhoneNumberBusinessInfo.ContactInfo.OperatingHours)
	agentPhoneNumberBusinessInfo.ContactInfo.Address = strings.TrimSpace(agentPhoneNumberBusinessInfo.ContactInfo.Address)
	if agentPhoneNumberBusinessInfo.ContactInfo.Email != "" && !helper.ValidateEmail(agentPhoneNumberBusinessInfo.ContactInfo.Email) {
		errors = append(errors, exception.NewInputException("business_contact_info.email", "invalid email"))
	}
	return errors
}

type AgentFAQ struct {
	ID        string `json:"id"`
	Question  string `json:"question"`
	Answer    string `json:"answer"`
	CreatedAt int64  `json:"created_at"`
}

type AgentFile struct {
	ID       string `json:"id"`
	FileName string `json:"file_name"`
}

type AgentWebsite struct {
	ID                  string   `json:"id"`
	URL                 string   `json:"url"`
	CrawlStatus         string   `json:"crawl_status"`
	CrawlError          string   `json:"crawl_error"`
	PageCrawled         int      `json:"pages_crawled"`
	LastCrawledAt       int64    `json:"last_crawled_at"`
	CreatedAt           int64    `json:"created_at"`
	IncludedSubDomains  []string `json:"included_sub_domains"`
	IncludedURLPatterns []string `json:"included_url_patterns"`
	ExcludedSubDomains  []string `json:"excluded_sub_domains"`
	ExcludedURLPatterns []string `json:"excluded_url_patterns"`
	SingleURLs          []string `json:"single_urls"`
}

func (website *AgentWebsite) Validate() []exception.InputException {
	website.URL = strings.TrimSpace(website.URL)
	var errors []exception.InputException
	if website.URL == "" {
		errors = append(errors, exception.NewInputException("url", "missing url"))
	}
	parsedURL, err := url.ParseRequestURI(website.URL)
	if err != nil || parsedURL.Scheme == "" || parsedURL.Host == "" || (parsedURL.Scheme != "http" && parsedURL.Scheme != "https") {
		errors = append(errors, exception.NewInputException("url", "must be a valid HTTP or HTTPS URL"))
	}
	return errors
}

type AgentSkill struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Skill       string `json:"skill"`
	Channel     string `json:"channel"`
	CreatedAt   int64  `json:"created_at"`
	Status      string `json:"status"`
}

func (agentSkill *AgentSkill) Validate() []exception.InputException {
	agentSkill.Title = strings.TrimSpace(agentSkill.Title)
	agentSkill.Description = strings.TrimSpace(agentSkill.Description)
	var errors []exception.InputException
	if agentSkill.Title == "" || agentSkill.Title[0] == '-' || agentSkill.Title[len(agentSkill.Title)-1] == '-' {
		errors = append(errors, exception.NewInputException("title", "must contain only lowercase letters, numbers, and hyphens, and must not start or end with a hyphen"))
	}
	if len(agentSkill.Title) > 64 {
		errors = append(errors, exception.NewInputException("title", "title must be within 64 characters"))
	}
	for _, character := range agentSkill.Title {
		if (character < 'a' || character > 'z') && (character < '0' || character > '9') && character != '-' {
			errors = append(errors, exception.NewInputException("title", "must contain only lowercase letters, numbers, and hyphens, and must not start or end with a hyphen"))
		}
	}
	if agentSkill.Description == "" {
		errors = append(errors, exception.NewInputException("description", "missing description"))
	}
	if agentSkill.Skill == "" {
		errors = append(errors, exception.NewInputException("skill", "missing skill"))
	}
	return nil
}

type AgentUISkillsResponse struct {
	Data []AgentUISkill `json:"data"`
}

type AgentUISkill struct {
	ID            string `json:"id"`
	Title         string `json:"title"`
	ComponentType string `json:"component_type"`
	Status        string `json:"status"`
	Instruction   string `json:"instruction"`
	CreatedAt     int64  `json:"created_at"`
}

func (agentUISkill *AgentUISkill) Validate() []exception.InputException {
	agentUISkill.ID = strings.TrimSpace(agentUISkill.ID)
	agentUISkill.Title = strings.TrimSpace(agentUISkill.Title)
	agentUISkill.ComponentType = strings.TrimSpace(agentUISkill.ComponentType)
	agentUISkill.Instruction = strings.TrimSpace(agentUISkill.Instruction)
	var errors []exception.InputException
	if agentUISkill.Title == "" || agentUISkill.Title[0] == '-' || agentUISkill.Title[len(agentUISkill.Title)-1] == '-' {
		errors = append(errors, exception.NewInputException("title", "must contain only lowercase letters, numbers, and hyphens, and must not start or end with a hyphen"))
	}
	for _, character := range agentUISkill.Title {
		if (character < 'a' || character > 'z') && (character < '0' || character > '9') && character != '-' {
			errors = append(errors, exception.NewInputException("title", "must contain only lowercase letters, numbers, and hyphens, and must not start or end with a hyphen"))
		}
	}
	if agentUISkill.ComponentType == "" {
		errors = append(errors, exception.NewInputException("component_type", "missing component type"))
	}
	if agentUISkill.Instruction == "" {
		errors = append(errors, exception.NewInputException("instruction", "missing instruction"))
	}
	return errors
}

type AgentTestResponse struct {
	MessageId        string `json:"message_id"`
	AgentResponse    string `json:"agent_response"`
	ConversationId   string `json:"conversation_id"`
	Timestamp        int64  `json:"timestamp"`
	HandoffReason    string `json:"handoff_reason"`
	NoResponseReason string `json:"no_response_reason"`
}

type AgentSetting struct {
	Rollout         AgentSettingRollout  `json:"rollout"`
	Handoff         AgentSettingHandoff  `json:"handoff"`
	Followup        AgentSettingFollowup `json:"followup"`
	NeverSayPhrases []string             `json:"never_say_phrases"`
	AIAudience      string               `json:"ai_audience"`
}

type AgentSettingRollout struct {
	Enabled bool `json:"enabled"`
}

type AgentSettingHandoff struct {
	Enabled          bool   `json:"enabled"`
	Message          string `json:"message"`
	MessageSelection string `json:"message_selection"`
}

type AgentSettingFollowup struct {
	Enabled                   bool   `json:"enabled"`
	FollowupIntervalInSeconds int    `json:"followup_interval_in_seconds"`
	Message                   string `json:"message"`
}

func (setting *AgentSetting) Validate() []exception.InputException {
	setting.AIAudience = "EVERYONE"
	setting.Handoff.MessageSelection = "CUSTOM"
	setting.Followup.Message = strings.TrimSpace(setting.Followup.Message)
	setting.Handoff.Message = strings.TrimSpace(setting.Handoff.Message)
	var errors []exception.InputException
	if setting.Followup.Enabled && setting.Followup.Message == "" {
		errors = append(errors, exception.NewInputException("followup.message", "missing followup message"))
	}
	if setting.Handoff.Enabled && setting.Handoff.Message == "" {
		errors = append(errors, exception.NewInputException("handoff.message", "missing handoff message"))
	}
	//300, 900, 1800, 3600, 7200, 28800, 86400
	followupIntervalSeconds := []int{300, 900, 1800, 3600, 7200, 28800, 86400}
	if setting.Followup.Enabled && !helper.Any(followupIntervalSeconds, func(second int) bool {
		return second == setting.Followup.FollowupIntervalInSeconds
	}) {
		errors = append(errors, exception.NewInputException("followup.followup_interval_in_seconds", "invalid followup interval seconds"))
	}
	return errors
}

type AgentConnector struct {
	ID                      string                                 `json:"id,omitempty"`
	Name                    string                                 `json:"name"`
	Description             string                                 `json:"description,omitempty"`
	BaseURL                 string                                 `json:"base_url"`
	ConnectorProtocol       string                                 `json:"connector_protocol"`
	MCPToolSync             *AgentConnectorMCPToolSync             `json:"mcp_tool_sync,omitempty"`
	AuthType                string                                 `json:"auth_type"`
	AuthConfig              *AgentConnectorAuthConfig              `json:"auth_config,omitempty"`
	MTLSConfig              *AgentConnectorMTLSConfig              `json:"mtls_config,omitempty"`
	ConnectionStatus        *AgentConnectorConnectionStatus        `json:"connection_status,omitempty"`
	UserAuthInjectionConfig *AgentConnectorUserAuthInjectionConfig `json:"user_auth_injection_config,omitempty"`
	RequiresCertificate     bool                                   `json:"requires_certificate"`
}

type AgentConnectorMCPToolSync struct {
	Status           string `json:"status"`
	LastAttemptedAt  int64  `json:"last_attempted_at"`
	LastSuccessfulAt int64  `json:"last_successful_at"`
	Fingerprint      string `json:"fingerprint"`
	ToolCount        int    `json:"tool_count"`
}

type AgentConnectorAuthConfig struct {
	OAuth2ClientCredentials *AgentConnectorOAuth2ClientCredentials `json:"oauth2_client_credentials,omitempty"`
	APIKey                  *AgentConnectorAPIKey                  `json:"api_key,omitempty"`
}

type AgentConnectorOAuth2ClientCredentials struct {
	TokenURL                string   `json:"token_url"`
	ScopesToRequest         []string `json:"scopes_to_request"`
	TokenRequestContentType string   `json:"token_request_content_type"`
	ClientId                string   `json:"client_id"`
	ClientSecret            string   `json:"client_secret"`
}

type AgentConnectorAPIKey struct {
	Headers     []AgentConnectorAPIKeyHeader `json:"headers,omitempty"`
	QueryParams []AgentConnectorAPIKeyHeader `json:"query_params,omitempty"`
	BodyParams  []map[string]any             `json:"body_params,omitempty"`
}

type AgentConnectorAPIKeyHeader struct {
	FieldName string `json:"field_name"`
	Value     string `json:"value"`
	Prefix    string `json:"prefix"`
}

type AgentConnectorMTLSConfig struct {
	HasCertificate    bool   `json:"has_certificate"`
	Fingerprint       string `json:"fingerprint"`
	ExpiresAt         int64  `json:"expires_at"`
	Subject           string `json:"subject"`
	ClientCertificate string `json:"client_certificate"`
	CACertificate     string `json:"ca_certificate"`
}

type AgentConnectorConnectionStatus struct {
	Status       string `json:"status"`
	ErrorMessage string `json:"error_message"`
}

type AgentConnectorUserAuthInjectionConfig struct {
	Location  string `json:"location"`
	FieldName string `json:"field_name"`
	Prefix    string `json:"prefix"`
}

type AgentConnectorLog struct {
	Data  []AgentConnectorLogData `json:"data"`
	Stats AgentConnectorStats     `json:"stats"`
}

type AgentConnectorLogData struct {
	EventTime       string `json:"event_time"`
	FailureCodeName string `json:"failure_code_name"`
	ErrorMessage    string `json:"error_message"`
	ToolName        string `json:"tool_name"`
	Occurences      int    `json:"occurrences"`
	LastSeen        string `json:"last_seen"`
}

type AgentConnectorStats struct {
	StartCount        int     `json:"start_count"`
	SuccessCount      int     `json:"success_count"`
	ExceptionCount    int     `json:"exception_count"`
	SuccessRate       string  `json:"success_date"`
	AvgLatencySeconds string  `json:"avg_latency_s"`
	P95LatencySeconds float32 `json:"p95_latency_s"`
	P99LatencySeconds float32 `json:"p99_latency_s"`
	TimeWindowSeconds int     `json:"time_window_seconds"`
}

// #region business agent

func (facebook *Facebook) CheckAgentEligibility(ctx context.Context, metaPhoneNumberId string, businessAccessToken string) (bool, error) {
	endpoint := fmt.Sprintf("%s/%s/agent_eligibility", facebook.baseURL, url.PathEscape(metaPhoneNumberId))
	var response AppAgentEligibilityResponse
	if err := facebook.doRequest(ctx, "check_agent_eligibility", http.MethodGet, endpoint, map[string]any{}, &response, businessAccessToken); err != nil {
		return false, err
	}
	return response.IsEligible, nil
}

func (facebook *Facebook) OnboardAgent(ctx context.Context, metaPhoneNumberId string, businessAccessToken string) (string, error) {
	endpoint := fmt.Sprintf("%s/%s/agent_onboarding/", facebook.baseURL, metaPhoneNumberId)
	var response AgentOnboardResponse
	if err := facebook.doRequest(ctx, "onboard_agent", http.MethodPost, endpoint, map[string]any{}, &response, businessAccessToken); err != nil {
		return "", err
	}
	return response.AgentId, nil
}

func (facebook *Facebook) OffboardAgent(ctx context.Context, metaPhoneNumberId string, businessAccessToken string) (string, error) {
	endpoint := fmt.Sprintf("%s/%s/delete_agent/", facebook.baseURL, metaPhoneNumberId)
	var response AgentOffboardResponse
	if err := facebook.doRequest(ctx, "offboard_agent", http.MethodDelete, endpoint, map[string]any{}, &response, businessAccessToken); err != nil {
		return "", err
	}
	return response.DeleteAgentId, nil
}

func (facebook *Facebook) ListAgentBudgets(ctx context.Context, metaBusinessPortfolioId string, businessAccessToken string) ([]AgentBudget, error) {
	endpoint := fmt.Sprintf("%s/%s/agent_budget/", facebook.baseURL, metaBusinessPortfolioId)
	var response AgentBugetsResponse
	if err := facebook.doRequest(ctx, "list_agent_budgets", http.MethodGet, endpoint, nil, &response, businessAccessToken); err != nil {
		return nil, err
	}
	return response.Budgets, nil
}

func (facebook *Facebook) UpdateAgentBudgets(ctx context.Context, metaBusinessPortfolioId string, businessAccessToken string, budgets []AgentBudget) ([]AgentBudget, error) {
	endpoint := fmt.Sprintf("%s/%s/agent_budget/", facebook.baseURL, metaBusinessPortfolioId)
	var payload map[string]any = make(map[string]any)
	payload["budgets"] = budgets
	var response AgentBugetsResponse
	if err := facebook.doRequest(ctx, "list_agent_budgets", http.MethodPost, endpoint, payload, &response, businessAccessToken); err != nil {
		return nil, err
	}
	return response.Budgets, nil
}

func (facebook *Facebook) GetAgentBusinessInfo(ctx context.Context, metaPhoneNumberId string, businessAccessToken string) (*AgentPhoneNumberBusinessInfo, error) {
	endpoint := fmt.Sprintf("%s/%s/agent_config/business_info", facebook.baseURL, metaPhoneNumberId)
	var response AgentPhoneNumberBusinessInfo
	if err := facebook.doRequest(ctx, "get_agent_business_info", http.MethodGet, endpoint, nil, &response, businessAccessToken); err != nil {
		return nil, err
	}
	return &response, nil
}

func (facebook *Facebook) UpdateAgentBusinessInfo(ctx context.Context, metaPhoneNumberId string, businessInfo *AgentPhoneNumberBusinessInfo, businessAccessToken string) error {
	endpoint := fmt.Sprintf("%s/%s/agent_config/business_info", facebook.baseURL, metaPhoneNumberId)
	if err := facebook.doRequest(ctx, "update_agent_business_info", http.MethodPut, endpoint, *businessInfo, nil, businessAccessToken); err != nil {
		return err
	}
	return nil
}

func (facebook *Facebook) ListAgentFAQs(ctx context.Context, metaPhoneNumberId string, businessAccessToken string) ([]AgentFAQ, error) {
	endpoint := fmt.Sprintf("%s/%s/agent_config/faq", facebook.baseURL, metaPhoneNumberId)
	var response []AgentFAQ
	if err := facebook.doRequest(ctx, "list_agent_faqs", http.MethodGet, endpoint, nil, &response, businessAccessToken); err != nil {
		return nil, err
	}
	return response, nil
}

func (facebook *Facebook) CreateAgentFAQ(ctx context.Context, metaPhoneNumberId string, businessAccessToken string, faq *AgentFAQ) (*AgentFAQ, error) {
	endpoint := fmt.Sprintf("%s/%s/agent_config/faq", facebook.baseURL, metaPhoneNumberId)
	var response AgentFAQ
	if err := facebook.doRequest(ctx, "create_agent_faq", http.MethodPost, endpoint, *faq, &response, businessAccessToken); err != nil {
		return nil, err
	}
	return &response, nil
}

func (facebook *Facebook) UpdateAgentFAQ(ctx context.Context, metaPhoneNumberId string, businessAccessToken string, faq *AgentFAQ) (*AgentFAQ, error) {
	endpoint := fmt.Sprintf("%s/%s/agent_config/faq/%s", facebook.baseURL, metaPhoneNumberId, faq.ID)
	var response AgentFAQ
	if err := facebook.doRequest(ctx, "update_agent_faq", http.MethodPut, endpoint, *faq, &response, businessAccessToken); err != nil {
		return nil, err
	}
	return &response, nil
}

func (facebook *Facebook) DeleteAgentFAQ(ctx context.Context, metaPhoneNumberId string, businessAccessToken string, id string) error {
	endpoint := fmt.Sprintf("%s/%s/agent_config/faq/%s", facebook.baseURL, metaPhoneNumberId, id)
	if err := facebook.doRequest(ctx, "delete_agent_faq", http.MethodDelete, endpoint, nil, nil, businessAccessToken); err != nil {
		return err
	}
	return nil
}

func (facebook *Facebook) ListAgentFiles(ctx context.Context, metaPhoneNumberId string, businessAccessToken string) ([]AgentFile, error) {
	endpoint := fmt.Sprintf("%s/%s/agent_config/files", facebook.baseURL, metaPhoneNumberId)
	var response []AgentFile
	if err := facebook.doRequest(ctx, "list_agent_files", http.MethodGet, endpoint, nil, &response, businessAccessToken); err != nil {
		return nil, err
	}
	return response, nil
}

func (facebook *Facebook) CreateAgentFile(ctx context.Context, metaPhoneNumberId string, fileName string, content []byte, businessAccessToken string) (*AgentFile, error) {
	if len(content) == 0 {
		return nil, fmt.Errorf("file content is required")
	}
	if len(content) > helper.MaxUploadFileSizeBytes {
		return nil, fmt.Errorf("file content exceeds the 15MB size limit")
	}
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	if err := writer.WriteField("file_name", fileName); err != nil {
		return nil, fmt.Errorf("failed to write agent file name: %w", err)
	}
	filePart, err := writer.CreateFormFile("file", fileName)
	if err != nil {
		return nil, fmt.Errorf("failed to create agent file part: %w", err)
	}
	if _, err := filePart.Write(content); err != nil {
		return nil, fmt.Errorf("failed to write agent file content: %w", err)
	}
	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("failed to close agent file form: %w", err)
	}
	endpoint := fmt.Sprintf("%s/%s/agent_config/files/", facebook.baseURL, metaPhoneNumberId)
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create agent file request: %w", err)
	}
	request.Header.Set("Authorization", "Bearer "+businessAccessToken)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	request.Header.Set("X-API-Version", "2.0.0")
	if userAgent := helper.GetUserAgent(ctx); userAgent != nil && strings.TrimSpace(*userAgent) != "" {
		request.Header.Set("User-Agent", *userAgent)
	}
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return nil, fmt.Errorf("agent file request failed: %w", err)
	}
	defer response.Body.Close()
	responseBody, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read agent file response: %w", err)
	}
	responseText := string(responseBody)
	requestBody := map[string]any{"file_name": fileName, "file_size": len(content)}
	if responseText != "" {
		if err := facebook.saveRawResponse("create_agent_file", endpoint, http.MethodPost, requestBody, responseText); err != nil {
			facebook.logger.Warnf("failed to save Facebook raw response: %v", err)
		}
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return nil, facebook.parseAPIError(response.StatusCode, &responseText, false)
	}
	var agentFile AgentFile
	if err := json.Unmarshal(responseBody, &agentFile); err != nil {
		return nil, fmt.Errorf("failed to parse agent file response: %w", err)
	}
	return &agentFile, nil
}

func (facebook *Facebook) DeleteAgentFile(ctx context.Context, metaPhoneNumberId string, businessAccessToken string, id string) error {
	endpoint := fmt.Sprintf("%s/%s/agent_config/files/%s", facebook.baseURL, metaPhoneNumberId, id)
	if err := facebook.doRequest(ctx, "delete_agent_file", http.MethodDelete, endpoint, nil, nil, businessAccessToken); err != nil {
		return err
	}
	return nil
}

func (facebook *Facebook) ListAgentWebsites(ctx context.Context, metaPhoneNumberId string, businessAccessToken string) ([]AgentWebsite, error) {
	endpoint := fmt.Sprintf("%s/%s/agent_config/websites", facebook.baseURL, metaPhoneNumberId)
	var response []AgentWebsite
	if err := facebook.doRequest(ctx, "list_agent_websites", http.MethodGet, endpoint, nil, &response, businessAccessToken); err != nil {
		return nil, err
	}
	return response, nil
}

func (facebook *Facebook) GetAgentWebsite(ctx context.Context, metaPhoneNumberId string, businessAccessToken string, id string) (*AgentWebsite, error) {
	endpoint := fmt.Sprintf("%s/%s/agent_config/websites/%s", facebook.baseURL, metaPhoneNumberId, id) ///{entity_id}/agent_config/websites/{website_id}
	var response AgentWebsite
	if err := facebook.doRequest(ctx, "get_agent_website", http.MethodGet, endpoint, nil, &response, businessAccessToken); err != nil {
		return nil, err
	}
	return &response, nil
}

func (facebook *Facebook) CreateAgentWebsite(ctx context.Context, metaPhoneNumberId string, businessAccessToken string, website *AgentWebsite) (*AgentWebsite, error) {
	endpoint := fmt.Sprintf("%s/%s/agent_config/websites", facebook.baseURL, metaPhoneNumberId)
	var response AgentWebsite
	if err := facebook.doRequest(ctx, "create_agent_website", http.MethodPost, endpoint, *website, &response, businessAccessToken); err != nil {
		return nil, err
	}
	return &response, nil
}

func (facebook *Facebook) UpdateAgentWebsite(ctx context.Context, metaPhoneNumberId string, businessAccessToken string, website *AgentWebsite) (*AgentWebsite, error) {
	endpoint := fmt.Sprintf("%s/%s/agent_config/websites/%s", facebook.baseURL, metaPhoneNumberId, website.ID)
	var response AgentWebsite
	if err := facebook.doRequest(ctx, "update_agent_website", http.MethodPut, endpoint, *website, &response, businessAccessToken); err != nil {
		return nil, err
	}
	return &response, nil
}

func (facebook *Facebook) DeleteAgentWebsite(ctx context.Context, metaPhoneNumberId string, businessAccessToken string, id string) error {
	endpoint := fmt.Sprintf("%s/%s/agent_config/websites/%s", facebook.baseURL, metaPhoneNumberId, id)
	if err := facebook.doRequest(ctx, "delete_agent_website", http.MethodDelete, endpoint, nil, nil, businessAccessToken); err != nil {
		return err
	}
	return nil
}

func (facebook *Facebook) ListAgentSkills(ctx context.Context, metaPhoneNumberId string, businessAccessToken string) ([]AgentSkill, error) {
	endpoint := fmt.Sprintf("%s/%s/agent_config/skills", facebook.baseURL, metaPhoneNumberId)
	var response []AgentSkill
	if err := facebook.doRequest(ctx, "list_agent_skills", http.MethodGet, endpoint, nil, &response, businessAccessToken); err != nil {
		return nil, err
	}
	return response, nil
}

func (facebook *Facebook) CreateAgentSkill(ctx context.Context, metaPhoneNumberId string, businessAccessToken string, skill *AgentSkill) (*AgentSkill, error) {
	endpoint := fmt.Sprintf("%s/%s/agent_config/skills", facebook.baseURL, metaPhoneNumberId)
	var response AgentSkill
	if err := facebook.doRequest(ctx, "create_agent_skill", http.MethodPost, endpoint, *skill, &response, businessAccessToken); err != nil {
		return nil, err
	}
	return &response, nil
}

func (facebook *Facebook) UpdateAgentSkill(ctx context.Context, metaPhoneNumberId string, businessAccessToken string, skill *AgentSkill) (*AgentSkill, error) {
	endpoint := fmt.Sprintf("%s/%s/agent_config/skills/%s", facebook.baseURL, metaPhoneNumberId, skill.ID)
	var response AgentSkill
	if err := facebook.doRequest(ctx, "update_agent_skill", http.MethodPut, endpoint, *skill, &response, businessAccessToken); err != nil {
		return nil, err
	}
	return &response, nil
}

func (facebook *Facebook) DeleteAgentSkill(ctx context.Context, metaPhoneNumberId string, businessAccessToken string, id string) error {
	endpoint := fmt.Sprintf("%s/%s/agent_config/skills/%s", facebook.baseURL, metaPhoneNumberId, id)
	if err := facebook.doRequest(ctx, "delete_agent_skill", http.MethodDelete, endpoint, nil, nil, businessAccessToken); err != nil {
		return err
	}
	return nil
}

func (facebook *Facebook) ListAgentUISkills(ctx context.Context, metaPhoneNumberId string, businessAccessToken string) ([]AgentUISkill, error) {
	endpoint := fmt.Sprintf("%s/%s/agent-ui-skills", facebook.baseURL, metaPhoneNumberId)
	var response AgentUISkillsResponse
	if err := facebook.doRequest(ctx, "list_agent_ui_skills", http.MethodGet, endpoint, nil, &response, businessAccessToken); err != nil {
		return nil, err
	}
	return response.Data, nil
}

func (facebook *Facebook) CreateAgentUISkill(ctx context.Context, metaPhoneNumberId string, businessAccessToken string, skill *AgentUISkill) (*AgentUISkill, error) {
	endpoint := fmt.Sprintf("%s/%s/agent-ui-skills", facebook.baseURL, metaPhoneNumberId)
	var response AgentUISkill
	if err := facebook.doRequest(ctx, "create_agent_ui_skill", http.MethodPost, endpoint, *skill, &response, businessAccessToken); err != nil {
		return nil, err
	}
	return &response, nil
}

func (facebook *Facebook) UpdateAgentUISkill(ctx context.Context, metaPhoneNumberId string, businessAccessToken string, skill *AgentUISkill) (*AgentUISkill, error) {
	endpoint := fmt.Sprintf("%s/%s/agent-ui-skills/%s", facebook.baseURL, metaPhoneNumberId, skill.ID)
	var response AgentUISkill
	if err := facebook.doRequest(ctx, "update_agent_ui_skill", http.MethodPut, endpoint, *skill, &response, businessAccessToken); err != nil {
		return nil, err
	}
	return &response, nil
}

func (facebook *Facebook) DeleteAgentUISkill(ctx context.Context, metaPhoneNumberId string, businessAccessToken string, id string) error {
	endpoint := fmt.Sprintf("%s/%s/agent-ui-skills/%s", facebook.baseURL, metaPhoneNumberId, id)
	if err := facebook.doRequest(ctx, "delete_agent_ui_skill", http.MethodDelete, endpoint, nil, nil, businessAccessToken); err != nil {
		return err
	}
	return nil
}

func (facebook *Facebook) ListAgentConnectors(ctx context.Context, metaPhoneNumberId string, businessAccessToken string) ([]AgentConnector, error) {
	endpoint := fmt.Sprintf("%s/%s/agent_connectors", facebook.baseURL, metaPhoneNumberId)
	var response []AgentConnector
	if err := facebook.doRequest(ctx, "list_agent_connectors", http.MethodGet, endpoint, nil, &response, businessAccessToken); err != nil {
		return nil, err
	}
	return response, nil
}

func (facebook *Facebook) CreateAgentConnector(ctx context.Context, metaPhoneNumberId string, businessAccessToken string, agentConnector *AgentConnector) (*AgentConnector, error) {
	endpoint := fmt.Sprintf("%s/%s/agent_connectors", facebook.baseURL, metaPhoneNumberId)
	var response AgentConnector
	if err := facebook.doRequest(ctx, "create_agent_connector", http.MethodPost, endpoint, *agentConnector, &response, businessAccessToken); err != nil {
		return nil, err
	}
	return &response, nil
}

func (facebook *Facebook) UpdateAgentConnector(ctx context.Context, metaPhoneNumberId string, businessAccessToken string, connector *AgentConnector) (*AgentConnector, error) {
	endpoint := fmt.Sprintf("%s/%s/agent_connectors/%s", facebook.baseURL, metaPhoneNumberId, connector.ID)
	var response AgentConnector
	if err := facebook.doRequest(ctx, "update_agent_connector", http.MethodPut, endpoint, *connector, &response, businessAccessToken); err != nil {
		return nil, err
	}
	return &response, nil
}

func (facebook *Facebook) DeleteAgentConnector(ctx context.Context, metaPhoneNumberId string, businessAccessToken string, id string) error {
	endpoint := fmt.Sprintf("%s/%s/agent_connectors/%s", facebook.baseURL, metaPhoneNumberId, id)
	if err := facebook.doRequest(ctx, "delete_agent_connector", http.MethodDelete, endpoint, nil, nil, businessAccessToken); err != nil {
		return err
	}
	return nil
}

func (facebook *Facebook) ListAgentConnectorLogs(ctx context.Context, metaPhoneNumberId string, connectorId string, businessAccessToken string) (*AgentConnectorLog, error) {
	endpoint := fmt.Sprintf("%s/%s/agent_connectors/%s/logs", facebook.baseURL, metaPhoneNumberId, connectorId)
	var response AgentConnectorLog
	if err := facebook.doRequest(ctx, "list_agent_connectors", http.MethodGet, endpoint, nil, &response, businessAccessToken); err != nil {
		return nil, err
	}
	return &response, nil
}

func (facebook *Facebook) TestAgent(ctx context.Context, metaPhoneNumberId string, businessAccessToken string, userMessage string, conversationId string) (*AgentTestResponse, error) {
	endpoint := fmt.Sprintf("%s/%s/agent_test", facebook.baseURL, metaPhoneNumberId)
	var response AgentTestResponse
	payload := map[string]any{
		"user_msg":        userMessage,
		"conversation_id": conversationId,
	}
	if err := facebook.doRequest(ctx, "test_agent", http.MethodPost, endpoint, payload, &response, businessAccessToken); err != nil {
		return nil, err
	}
	return &response, nil
}

func (facebook *Facebook) GetSetting(ctx context.Context, metaPhoneNumberId string, businessAccessToken string) (*AgentSetting, error) {
	endpoint := fmt.Sprintf("%s/%s/agent_config/settings", facebook.baseURL, metaPhoneNumberId)
	var response []AgentSetting
	if err := facebook.doRequest(ctx, "get_setting", http.MethodGet, endpoint, nil, &response, businessAccessToken); err != nil {
		return nil, err
	}
	if len(response) > 0 {
		return &response[0], nil
	}
	return nil, nil
}

func (facebook *Facebook) UpdateSetting(ctx context.Context, metaPhoneNumberId string, businessAccessToken string, setting *AgentSetting) (*AgentSetting, error) {
	endpoint := fmt.Sprintf("%s/%s/agent_config/settings", facebook.baseURL, metaPhoneNumberId)
	var response AgentSetting
	if err := facebook.doRequest(ctx, "update_setting", http.MethodPut, endpoint, setting, &response, businessAccessToken); err != nil {
		return nil, err
	}
	return &response, nil
}

func (facebook *Facebook) TurnAgentOnOff(ctx context.Context, metaPhoneNumberId string, businessAccessToken string, on bool) error {
	endpoint := fmt.Sprintf("%s/%s/agent_config/settings", facebook.baseURL, metaPhoneNumberId)
	m := map[string]any{
		"rollout": map[string]any{
			"enabled": on,
		},
	}
	var response AgentSetting
	if err := facebook.doRequest(ctx, "update_setting", http.MethodPut, endpoint, m, &response, businessAccessToken); err != nil {
		return err
	}
	if response.Rollout.Enabled != on {
		return fmt.Errorf("response from Facebook update setting api rollout enabled is not same as %v", on)
	}
	return nil
}

func (facebook *Facebook) PassControl(ctx context.Context, metaPhoneNumberId string, waId string, businessAccessToken string, toAgent bool) error {
	endpoint := fmt.Sprintf("%s/business/whatsapp/phone_numbers/%s/thread_control", facebook.baseURL, metaPhoneNumberId)
	body := map[string]any{
		"messaging_product": "whatsapp",
		"to":                waId,
	}
	if toAgent {
		body["action"] = "release"
	} else {
		body["action"] = "take"
	}
	if err := facebook.doRequest(ctx, "pass_control", http.MethodPost, endpoint, body, nil, businessAccessToken); err != nil {
		return err
	}
	return nil
}

// #endregion

func (facebook *Facebook) doRequest(ctx context.Context, requestType string, method string, endpoint string, payload any, target any, businessAccessToken string) error {
	businessAccessToken = strings.TrimSpace(businessAccessToken)
	headers := map[string]string{}
	if businessAccessToken != "" {
		headers["Authorization"] = "Bearer " + businessAccessToken
	}
	if userAgent := helper.GetUserAgent(ctx); userAgent != nil && strings.TrimSpace(*userAgent) != "" {
		headers["User-Agent"] = *userAgent
	}
	// skip those still in 1.0.0
	var v1 bool
	if requestType != "pass_control" {
		headers["X-API-Version"] = "2.0.0"
	} else {
		headers["X-API-Version"] = "1.0.0"
		v1 = true
	}
	var bodyMap *map[string]any
	if payload != nil {
		data, err := json.Marshal(payload)
		if err != nil {
			return fmt.Errorf("Facebook request failed to marshal request payload: %w", err)
		}
		requestBody := map[string]any{}
		if err := json.Unmarshal(data, &requestBody); err != nil {
			return fmt.Errorf("Facebook request failed to convert request payload: %w", err)
		}
		bodyMap = &requestBody
	}
	statusCode, responseBody, _, err := helper.RequestHTTP(ctx, endpoint, method, &headers, bodyMap)
	if err != nil {
		return err
	}
	if responseBody != nil && strings.TrimSpace(*responseBody) != "" {
		if err := facebook.saveRawResponse(requestType, endpoint, method, bodyMap, *responseBody); err != nil {
			facebook.logger.Warnf("failed to save Facebook raw response: %v", err)
		}
	}
	if statusCode < 200 || statusCode >= 300 {
		// if statusCode == http.StatusTooManyRequests {
		// 	if retryAfterSeconds := maxEstimatedTimeToRegainAccess(usage); retryAfterSeconds > 0 {
		// 		return &WhatsAppRateLimitError{RetryAfterSeconds: retryAfterSeconds}
		// 	}
		// }
		return facebook.parseAPIError(statusCode, responseBody, v1)
	}
	if target == nil || responseBody == nil || strings.TrimSpace(*responseBody) == "" {
		return nil
	}
	if err := json.Unmarshal([]byte(*responseBody), target); err != nil {
		return fmt.Errorf("Facebook request failed to parse response: %w", err)
	}
	return nil
}

func (facebook *Facebook) parseAPIError(statusCode int, responseBody *string, v1 bool) error {
	if responseBody == nil || strings.TrimSpace(*responseBody) == "" {
		return fmt.Errorf("Facebook API request failed with status %d", statusCode)
	}
	if v1 {
		parsed, err := helper.DeserializeJSON[*APIErrorResponseV1](*responseBody)
		if err != nil || parsed == nil {
			return fmt.Errorf("Facebook API request failed with status %d: %s", statusCode, strings.TrimSpace(*responseBody))
		}
		return *parsed
	} else {
		parsed, err := helper.DeserializeJSON[*APIErrorResponse](*responseBody)
		if err != nil || parsed == nil {
			return fmt.Errorf("Facebook API request failed with status %d: %s", statusCode, strings.TrimSpace(*responseBody))
		}
		return *parsed
	}
}

type APIErrorResponse struct {
	Title   string `json:"title"`
	Detail  string `json:"detail"`
	Type    string `json:"type"`
	Status  int    `json:"status"`
	TraceId string `json:"fbtrace_id"`
}

func (apiErrorResponse *APIErrorResponse) Error() string {
	return fmt.Sprintf("Facebook API Error: %s\n%s", apiErrorResponse.Title, apiErrorResponse.Detail)
}

type APIErrorResponseV1 struct {
	ErrorV1 APIErrorResponseV1Error `json:"error"`
}

//	{
//	     "code": 1,
//	     "error_subcode": 2494180,
//	     "error_user_msg": "Only the current thread owner can pass or release control",
//	     "error_user_title": "Caller is not the thread owner",
//	     "fbtrace_id": "A27SkfXzFkQpEn_q9jvAPXJ",
//	     "is_transient": false,
//	     "message": "An unknown error occurred",
//	     "type": "OAuthException"
//	   }
type APIErrorResponseV1Error struct {
	Code         int    `json:"code"`
	ErrorSubcode int64  `json:"error_subcode"`
	UserMessage  string `json:"error_user_msg"`
	UserTitle    string `json:"error_user_title"`
	Message      string `json:"message"`
	Type         string `json:"type"`
	FBTraceId    string `json:"fbtrace_id"`
	IsTransient  bool   `json:"is_transient"`
}

func (apiErrorResponseV1 APIErrorResponseV1) Error() string {
	var items []string
	if apiErrorResponseV1.ErrorV1.UserTitle != "" {
		items = append(items, apiErrorResponseV1.ErrorV1.UserTitle)
	}
	if apiErrorResponseV1.ErrorV1.UserMessage != "" {
		items = append(items, apiErrorResponseV1.ErrorV1.UserMessage)
	}
	if len(items) == 0 {
		return apiErrorResponseV1.ErrorV1.Message
	}
	return strings.Join(items, "\n")
}

func (facebook *Facebook) saveRawResponse(requestType string, endpoint string, method string, requestBody any, responseBody string) error {
	if cfg.Default().Site.Environment != types.EnvironmentDevelop {
		return nil
	}
	if err := os.MkdirAll("files/facebook", 0755); err != nil {
		return fmt.Errorf("failed to create raw response directory: %w", err)
	}
	filename := fmt.Sprintf("%s.json", requestType)
	// if after := requestURLAfterCursor(endpoint); after != "" {
	// 	cursorHash := sha256.Sum256([]byte(after))
	// 	filename = fmt.Sprintf("%s-after-%x.json", requestType, cursorHash[:8])
	// }
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
	return helper.WriteToFile(string(data), filepath.Join("files/facebook", filename))
}
