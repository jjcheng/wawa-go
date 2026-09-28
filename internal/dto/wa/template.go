package dto_wa

import (
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"net/url"
	"strconv"
	"strings"

	"github.com/jjcheng/wawa-go/internal/cfg"
	dao_customer "github.com/jjcheng/wawa-go/internal/dao/customer"
	"github.com/jjcheng/wawa-go/internal/exception"
	"github.com/jjcheng/wawa-go/internal/helper"
	"github.com/jjcheng/wawa-go/internal/types"
)

// #region template
type Template struct {
	TemplateBase
	ID                  string                 `json:"id"`
	Status              types.WATemplateStatus `json:"status"`
	QualityScore        map[string]any         `json:"quality_score,omitempty"`
	RejectedReason      string                 `json:"rejected_reason,omitempty"`
	PreviousCategory    string                 `json:"previous_category,omitempty"`
	MetaEditTemplateUrl string                 `json:"meta_edit_template_url"`
	ByAPI               bool                   `json:"by_api"`
}

func (template *Template) GetMetaEditTemplateUrl(metaBusinessPortfolioId string, wabaId string) string {
	return fmt.Sprintf("https://business.facebook.com/latest/whatsapp_manager/message_templates/?business_id=%s&tab=message-templates&childRoute=CAPI&id=%s&nav_ref=whatsapp_manager&asset_id=%s", metaBusinessPortfolioId, template.ID, wabaId)
}

type TemplateBase struct {
	Name            string                          `json:"name"`
	Category        types.WATemplateCategory        `json:"category"`
	Language        string                          `json:"language"`
	ParameterFormat types.WATemplateParameterFormat `json:"parameter_format,omitempty"`
	Components      []TemplateComponent             `json:"components,omitempty"`
	// lazy loaded
	PreviewHTML     string                  `json:"preview_html,omitempty"`
	RawHTML         string                  `json:"raw_html,omitempty"`
	PreviewDarkHTML string                  `json:"preview_dark_html,omitempty"`
	RawDarkHTML     string                  `json:"raw_dark_html,omitempty"`
	SendComponents  []SendTemplateComponent `json:"send_components"`
}

func (templateBase *TemplateBase) ByAPI() bool {
	return strings.HasPrefix(templateBase.Name, "api_")
}

func (templateBase *TemplateBase) Validate() []exception.InputException {
	templateBase.Name = strings.TrimSpace(templateBase.Name)
	templateBase.Language = strings.TrimSpace(templateBase.Language)
	errors := []exception.InputException{}
	if templateBase.Name == "" {
		errors = append(errors, exception.NewInputException("name", "missing template name"))
	}
	if templateBase.Language == "" {
		errors = append(errors, exception.NewInputException("language", "missing template language"))
	}
	if templateBase.Category == "" {
		errors = append(errors, exception.NewInputException("category", "missing template category"))
	}
	if len(templateBase.Components) == 0 {
		errors = append(errors, exception.NewInputException("components", "missing template components"))
	}
	return errors
}

func (templateBase TemplateBase) Payload() map[string]any {
	return map[string]any{
		"name":       templateBase.Name,
		"language":   templateBase.Language,
		"category":   templateBase.Category,
		"components": templateBase.Components,
	}
}

// # region preview

type templateTheme struct {
	CanvasBackground string
	CardBackground   string
	CardBorder       string
	TextPrimary      string
	TextSecondary    string
	MutedBackground  string
	ActionBackground string
	ActionText       string
	ActionBorder     string
}

func whatsappTemplateTheme(dark bool) templateTheme {
	if dark {
		return templateTheme{
			CanvasBackground: "#0b141a",
			CardBackground:   "#1f2c34",
			CardBorder:       "#374248",
			TextPrimary:      "#e9edef",
			TextSecondary:    "#8696a0",
			MutedBackground:  "#2a3942",
			ActionBackground: "#202c33",
			ActionText:       "#53bdeb",
			ActionBorder:     "#374248",
		}
	}
	return templateTheme{
		CanvasBackground: "#efeae2",
		CardBackground:   "#ffffff",
		CardBorder:       "#d9dbe1",
		TextPrimary:      "#111b21",
		TextSecondary:    "#667781",
		MutedBackground:  "#f0f2f5",
		ActionBackground: "#ffffff",
		ActionText:       "#00a884",
		ActionBorder:     "#e9edef",
	}
}

func (template *Template) HTML(withExample bool, dark bool) string {
	theme := whatsappTemplateTheme(dark)
	var html strings.Builder
	for _, component := range template.Components {
		componentHtml, err := component.HTML(withExample, dark)
		if err != nil {
			fmt.Fprintf(&html, "%s ERROR: %s", component.Type, err.Error())
		} else {
			html.WriteString(componentHtml)
		}
	}
	return fmt.Sprintf("<div style='box-sizing:border-box;width:100%%;max-width:360px;padding:12px 12px 20px 12px;background:%s;font-family:Arial,sans-serif;'><div style='position:relative;overflow:hidden;width:100%%;max-width:330px;background:%s;border:1px solid %s;border-radius:8px 8px 8px 8px;color:%s;'>%s</div></div>", theme.CanvasBackground, theme.CardBackground, theme.CardBorder, theme.TextPrimary, html.String())
}

func (template *Template) ApplySendTemplate(sendTemplate SendTemplate) {
	for _, sendComponent := range sendTemplate.Components {
		switch strings.ToLower(sendComponent.Type) {
		case "header", "body":
			component := template.component(sendComponent.Type)
			if component == nil {
				continue
			}
			if strings.EqualFold(sendComponent.Type, "header") && applyHeaderMedia(component, sendComponent.Parameters) {
				continue
			}
			applyTextExamples(component, sendComponent.Parameters, template.ParameterFormat)
		case "button":
			template.applyButton(sendComponent)
		}
	}
}

func (template *Template) component(componentType string) *TemplateComponent {
	for i := range template.Components {
		if strings.EqualFold(string(template.Components[i].Type), componentType) {
			return &template.Components[i]
		}
	}
	return nil
}

func applyHeaderMedia(component *TemplateComponent, parameters []SendTemplateParameter) bool {
	for _, parameter := range parameters {
		var mediaURL string
		switch strings.ToLower(parameter.Type) {
		case "image":
			if parameter.Image != nil {
				mediaURL = parameter.Image.Link
			}
		case "video":
			if parameter.Video != nil {
				mediaURL = parameter.Video.Link
			}
		case "document":
			if parameter.Document != nil {
				mediaURL = parameter.Document.Link
			}
		case "location":
			if parameter.Location != nil {
				if component.Example == nil {
					component.Example = &TemplateComponentExample{}
				}
				component.Example.HeaderLocation = parameter.Location
				return true
			}
		}
		if mediaURL != "" {
			if component.Example == nil {
				component.Example = &TemplateComponentExample{}
			}
			component.Example.HeaderHandle = []string{mediaURL}
			return true
		}
	}
	return false
}

func applyTextExamples(component *TemplateComponent, parameters []SendTemplateParameter, parameterFormat types.WATemplateParameterFormat) {
	values := make([]string, 0, len(parameters))
	names := make([]string, 0, len(parameters))
	isNamed := parameterFormat == types.WATemplateParameterFormatNamed
	for index, parameter := range parameters {
		value, ok := sendTemplateParameterText(parameter)
		if !ok {
			continue
		}
		values = append(values, value)
		name := parameter.ParameterName
		if name != "" {
			isNamed = true
		}
		if name == "" && component.Example != nil {
			if component.Type == types.WATemplateComponentTypeHeader && index < len(component.Example.HeaderTextNamedParams) {
				name = component.Example.HeaderTextNamedParams[index].ParamName
				isNamed = true
			} else if component.Type == types.WATemplateComponentTypeBody && index < len(component.Example.BodyTextNamedParams) {
				name = component.Example.BodyTextNamedParams[index].ParamName
				isNamed = true
			}
		}
		names = append(names, name)
	}
	if len(values) == 0 {
		return
	}
	if component.Example == nil {
		component.Example = &TemplateComponentExample{}
	}
	if isNamed {
		namedValues := make([]TemplateComponentTextNamedParam, 0, len(values))
		for i, value := range values {
			namedValues = append(namedValues, TemplateComponentTextNamedParam{ParamName: names[i], Example: value})
		}
		if component.Type == types.WATemplateComponentTypeHeader {
			component.Example.HeaderText = nil
			component.Example.HeaderTextNamedParams = namedValues
		} else {
			component.Example.BodyText = nil
			component.Example.BodyTextNamedParams = namedValues
		}
		return
	}
	if component.Type == types.WATemplateComponentTypeHeader {
		component.Example.HeaderTextNamedParams = nil
		component.Example.HeaderText = values
	} else {
		component.Example.BodyTextNamedParams = nil
		component.Example.BodyText = [][]string{values}
	}
}

func sendTemplateParameterText(parameter SendTemplateParameter) (string, bool) {
	switch strings.ToLower(parameter.Type) {
	case "text":
		return parameter.Text, true
	case "currency":
		if parameter.Currency != nil {
			return parameter.Currency.FallbackValue, true
		}
	case "date_time":
		if parameter.DateTime != nil {
			return parameter.DateTime.FallbackValue, true
		}
	}
	return "", false
}

func (template *Template) applyButton(sendComponent SendTemplateComponent) {
	buttonIndex, err := strconv.Atoi(sendComponent.Index)
	if err != nil || buttonIndex < 0 {
		return
	}
	buttons := template.component(string(types.WATemplateComponentTypeButtons))
	if buttons == nil || buttonIndex >= len(buttons.Buttons) {
		return
	}
	for _, parameter := range sendComponent.Parameters {
		if strings.EqualFold(sendComponent.SubType, "url") && strings.EqualFold(parameter.Type, "text") {
			buttons.Buttons[buttonIndex].Example = []string{parameter.Text}
			buttons.Buttons[buttonIndex].Url = strings.Replace(buttons.Buttons[buttonIndex].Url, "{{1}}", parameter.Text, 1)
			return
		}
	}
}

func (templateComponent *TemplateComponent) HTML(withExample bool, dark bool) (string, error) {
	theme := whatsappTemplateTheme(dark)
	switch templateComponent.Type {
	case types.WATemplateComponentTypeHeader:
		switch templateComponent.Format {
		case types.WATemplateComponentFormatText:
			headerText := templateComponent.Text
			if withExample {
				if templateComponent.Example != nil {
					if len(templateComponent.Example.HeaderText) > 0 {
						for i, text := range templateComponent.Example.HeaderText {
							position := i + 1
							headerText = strings.Replace(headerText, fmt.Sprintf("{{%d}}", position), text, 1)
						}
					} else if len(templateComponent.Example.HeaderTextNamedParams) > 0 {
						for _, param := range templateComponent.Example.HeaderTextNamedParams {
							headerText = strings.ReplaceAll(headerText, fmt.Sprintf("{{%s}}", param.ParamName), param.Example)
						}
					}
				}
				if strings.Contains(headerText, "{{") {
					return "", errors.New("no header_text or header_text_named_params in example")
				}
			}
			return fmt.Sprintf("<div style='padding:8px 9px 0;font-size:14.2px;font-weight:600;color:%s;line-height:19px;'>%s</div>", theme.TextPrimary, formatWhatsAppText(headerText)), nil
		case types.WATemplateComponentFormatImage:
			if templateComponent.Example == nil || len(templateComponent.Example.HeaderHandle) == 0 {
				return "", errors.New("no example or no header_handle")
			}
			return fmt.Sprintf("<div style='margin:0;overflow:hidden;border-radius:0;line-height:0;'><img src='%s' alt='Template header' style='display:block;width:100%%;height:auto;max-height:260px;object-fit:cover;border-radius:0;'></div>", templateComponent.Example.HeaderHandle[0]), nil
		case types.WATemplateComponentFormatVideo:
			if templateComponent.Example == nil || len(templateComponent.Example.HeaderHandle) == 0 {
				return "", errors.New("no example or no header_handle")
			}
			return fmt.Sprintf("<div style='padding:0;'><video controls preload='metadata' playsinline src='%s' style='display:block;width:100%%;max-height:260px;object-fit:cover;border-radius:0;background:#0b141a;'></video></div>", templateComponent.Example.HeaderHandle[0]), nil
		case types.WATemplateComponentFormatDocument:
			if templateComponent.Example == nil || len(templateComponent.Example.HeaderHandle) == 0 {
				return "", errors.New("no example or no header_handle")
			}
			documentURL := templateComponent.Example.HeaderHandle[0]
			if strings.HasSuffix(strings.ToLower(documentURL), ".pdf") {
				return fmt.Sprintf("<div style='padding:0;'><iframe src='%s' title='PDF preview' loading='lazy' style='display:block;width:100%%;height:220px;border:0;border-radius:0;background:%s;'></iframe><a href='%s' target='_blank' rel='noopener noreferrer' style='display:flex;align-items:center;gap:8px;margin-top:2px;padding:9px 10px;border-radius:0;background:%s;color:%s;text-decoration:none;font-size:13px;'><span style='font-size:20px;'>📄</span><span>Open PDF</span></a></div>", documentURL, theme.MutedBackground, documentURL, theme.MutedBackground, theme.TextPrimary), nil
			}
			documentType := classifyDocumentType(documentURL)
			documentIcon := documentIconForType(documentType)
			return fmt.Sprintf("<div style='padding:0;'><a href='%s' target='_blank' rel='noopener noreferrer' style='display:flex;align-items:center;gap:10px;padding:12px;border-radius:0;background:%s;color:%s;text-decoration:none;font-size:13px;'><span style='font-size:26px;line-height:1;'>%s</span><span style='overflow:hidden;text-overflow:ellipsis;white-space:nowrap;'>%s document</span></a></div>", documentURL, theme.MutedBackground, theme.TextPrimary, documentIcon, strings.ToUpper(documentType)), nil
		case types.WATemplateComponentFormatLocation:
			location := &SendTemplateParameterLocation{Latitude: 1.3521, Longitude: 103.844, Name: "Location name", Address: "Location address"}
			if templateComponent.Example != nil && templateComponent.Example.HeaderLocation != nil {
				location = templateComponent.Example.HeaderLocation
			}
			mapQuery := url.Values{}
			mapQuery.Set("center", fmt.Sprintf("%f,%f", location.Latitude, location.Longitude))
			mapQuery.Set("zoom", "15")
			mapQuery.Set("size", "450x300")
			mapQuery.Set("scale", "2")
			mapQuery.Set("maptype", "roadmap")
			mapQuery.Set("markers", fmt.Sprintf("color:red|%f,%f", location.Latitude, location.Longitude))
			mapQuery.Set("key", cfg.Default().Site.GoogleMapAPIKey)
			if dark {
				mapQuery.Set("map_id", cfg.Default().Site.GoogleDarkMapID)
			} else {
				mapQuery.Set("map_id", cfg.Default().Site.GoogleLightMapID)
			}
			mapURL := "https://maps.googleapis.com/maps/api/staticmap?" + mapQuery.Encode()
			return fmt.Sprintf("<div style='padding:0;'><img src='%s' alt='Location map' style='display:block;width:100%%;height:170px;object-fit:cover;border-radius:0;'><div style='padding:8px 9px;background:%s;border-radius:0;'><div style='font-weight:600;font-size:13.5px;line-height:18px;color:%s;'>%s</div><div style='margin-top:2px;font-size:12px;line-height:16px;color:%s;'>%s</div></div></div>", mapURL, theme.MutedBackground, theme.TextPrimary, html.EscapeString(location.Name), theme.TextSecondary, html.EscapeString(location.Address)), nil
		default:
			return "", errors.New("unsupported header format")
		}

	case types.WATemplateComponentTypeBody:
		bodyText := templateComponent.Text
		if withExample {
			if templateComponent.Example != nil {
				if len(templateComponent.Example.BodyText) > 0 && len(templateComponent.Example.BodyText[0]) > 0 {
					for i, text := range templateComponent.Example.BodyText[0] {
						position := i + 1
						bodyText = strings.Replace(bodyText, fmt.Sprintf("{{%d}}", position), text, 1)
					}
				} else if len(templateComponent.Example.BodyTextNamedParams) > 0 {
					for _, param := range templateComponent.Example.BodyTextNamedParams {
						bodyText = strings.ReplaceAll(bodyText, fmt.Sprintf("{{%s}}", param.ParamName), param.Example)
					}
				}
			}
			if strings.Contains(bodyText, "{{") {
				return "", errors.New("no body_text or body_text_named_params in example")
			}
		}
		return fmt.Sprintf("<div style='padding:6px 9px 8px;color:%s;font-size:14.2px;line-height:19px;white-space:pre-wrap;overflow-wrap:anywhere;'>%s</div>", theme.TextPrimary, formatWhatsAppText(bodyText)), nil
	case types.WATemplateComponentTypeFooter:
		return fmt.Sprintf("<div style='padding:4px 9px 7px;color:%s;font-size:12px;line-height:16px;white-space:pre-wrap;overflow-wrap:anywhere;'>%s</div>", theme.TextSecondary, formatWhatsAppText(templateComponent.Text)), nil
	case types.WATemplateComponentTypeButtons:
		if len(templateComponent.Buttons) == 0 {
			return "", errors.New("no buttons in component")
		}
		buttonStyle := fmt.Sprintf("display:block;width:100%%;padding:10px 12px;border:0;border-top:1px solid %s;background:%s;color:%s;font-size:14px;font-weight:500;line-height:20px;text-align:center;", theme.ActionBorder, theme.ActionBackground, theme.ActionText)
		var buttonHTML []string
		for _, button := range templateComponent.Buttons {
			switch button.Type {
			case types.WATemplateButtonTypeURL:
				buttonHTML = append(buttonHTML, fmt.Sprintf("<a href='%s' target='_blank' rel='noopener noreferrer' style='%s'>↗&nbsp; %s</a>", button.Url, buttonStyle, button.Text))
			case types.WATemplateButtonTypePhoneNumber:
				buttonHTML = append(buttonHTML, fmt.Sprintf("<a href='tel:%s' style='%s'>☎&nbsp; %s</a>", button.PhoneNumber, buttonStyle, button.Text))
			case types.WATemplateButtonTypeQuickReply:
				buttonHTML = append(buttonHTML, fmt.Sprintf("<button type='button' style='%scursor:pointer;'>↩&nbsp; %s</button>", buttonStyle, button.Text))
			case types.WATemplateButtonTypeVoiceCall:
				buttonHTML = append(buttonHTML, fmt.Sprintf("<button type='button' style='%scursor:pointer;'>☎&nbsp; %s</button>", buttonStyle, button.Text))
			case types.WATemplateButtonTypeCopyCode:
				buttonHTML = append(buttonHTML, fmt.Sprintf("<button type='button' style='%scursor:pointer;'>⧉&nbsp; %s</button>", buttonStyle, button.Text))
			default:
				buttonHTML = append(buttonHTML, fmt.Sprintf("<button type='button' style='%scursor:pointer;'>%s</button>", buttonStyle, button.Text))
			}
		}
		return fmt.Sprintf("<div style='margin-top:3px;background:%s;'>%s</div>", theme.ActionBackground, strings.Join(buttonHTML, "")), nil
	case types.WATemplateComponentTypeCallPermissionRequest:
		return fmt.Sprintf("<div style='margin:9px 9px 0;padding:16px;background:%s;display:flex;align-items:flex-start;gap:12px;'><div style='box-sizing:border-box;flex:0 0 58px;width:58px;height:58px;border-radius:50%%;background:%s;display:flex;align-items:center;justify-content:center;color:%s;'><svg viewBox='0 0 24 24' width='25' height='25' aria-hidden='true' style='display:block;fill:currentColor;'><path d='M6.62 10.79a15.5 15.5 0 0 0 6.59 6.59l2.2-2.2a1 1 0 0 1 1.02-.24c1.12.37 2.33.57 3.57.57a1 1 0 0 1 1 1V20a1 1 0 0 1-1 1C10.61 21 3 13.39 3 4a1 1 0 0 1 1-1h3.5a1 1 0 0 1 1 1c0 1.25.2 2.45.57 3.57a1 1 0 0 1-.25 1.02l-2.2 2.2Z'></path></svg></div><div style='min-width:0;flex:1;padding-top:2px;'><div style='color:%s;font-size:16px;font-weight:700;line-height:21px;'>Can {BIZ_NAME} call you?</div><div style='margin-top:3px;color:%s;font-size:15px;font-weight:400;line-height:22px;'>You can update your<br>preference at any time<br>in the business profile. <span style='display:inline-block;margin-left:8px;font-size:12px;line-height:16px;white-space:nowrap;'>04:18</span></div></div></div><div style='height:54px;display:flex;align-items:center;justify-content:center;gap:12px;background:%s;color:%s;font-size:15px;font-weight:500;line-height:20px;border-top:1px solid %s;'>Choose preference<span aria-hidden='true' style='display:inline-block;width:10px;height:10px;border-right:2px solid %s;border-bottom:2px solid %s;transform:rotate(45deg) translateY(-3px);'></span></div>", theme.MutedBackground, theme.ActionBackground, theme.TextPrimary, theme.TextPrimary, theme.TextSecondary, theme.ActionBackground, theme.ActionText, theme.ActionBorder, theme.ActionText, theme.ActionText), nil

	default:
		return "", errors.New("unsupported template component type")
	}
}

// #endregion

// # region list

type TemplateListResponse struct {
	Data   []Template      `json:"data"`
	Paging *TemplatePaging `json:"paging,omitempty"`
}

type TemplatePaging struct {
	Previous string                 `json:"previous"`
	Next     string                 `json:"next"`
	Cursors  *TemplatePagingCursors `json:"cursors,omitempty"`
}

type TemplatePagingCursors struct {
	Before string `json:"before,omitempty"`
	After  string `json:"after,omitempty"`
}

// # end region

// #region usage

type TemplateAnalytics struct {
	WABATimezone   string                       `json:"waba_timezone,omitempty"`
	Granularity    string                       `json:"granularity,omitempty"`
	ProductType    string                       `json:"product_type,omitempty"`
	DataPoints     []TemplateAnalyticsDataPoint `json:"data_points"`
	TotalSent      int                          `json:"total_sent"`
	TotalDelivered int                          `json:"total_delivered"`
	TotalRead      int                          `json:"total_read"`
	TotalClicked   int                          `json:"total_clicked"`
}

type TemplateAnalyticsDataPoint struct {
	TemplateID string                        `json:"template_id"`
	Start      int64                         `json:"start"`
	End        int64                         `json:"end"`
	Sent       int64                         `json:"sent,omitempty"`
	Delivered  int64                         `json:"delivered,omitempty"`
	Read       int64                         `json:"read,omitempty"`
	Clicked    []TemplateAnalyticsClickEntry `json:"clicked,omitempty"`
}

type TemplateAnalyticsClickEntry struct {
	ButtonContent string `json:"button_content"`
	Count         int64  `json:"count"`
	Type          string `json:"type"`
}

type TemplateAnalyticsListResponse struct {
	Data   []TemplateAnalytics `json:"data"`
	Paging *TemplatePaging     `json:"paging,omitempty"`
}

// #endregion

// #region create

type TemplateComponent struct {
	Format  types.WATemplateComponentFormat `json:"format" val:"required" description:"TEXT, IMAGE, VIDEO, DOCUMENT, LOCATION"`
	Text    string                          `json:"text,omitempty" description:"only if format = TEXT"`
	Type    types.WATemplateComponentType   `json:"type" description:"HEADER, BODY, FOOTER, BUTTONS"`
	Example *TemplateComponentExample       `json:"example,omitempty" description:"only present if type is HEADER, BODY"`
	Buttons []TemplateComponentButton       `json:"buttons,omitempty" description:"buttons below the message"`
}

type TemplateComponentExample struct {
	HeaderTextNamedParams []TemplateComponentTextNamedParam `json:"header_text_named_params,omitempty" description:"present if type is HEADER"`
	BodyTextNamedParams   []TemplateComponentTextNamedParam `json:"body_text_named_params,omitempty" description:"present if type is body"`
	HeaderHandle          []string                          `json:"header_handle,omitempty" description:"present if format is IMAGE, VIDEO, DOCUMENT"`
	HeaderText            []string                          `json:"header_text,omitempty" description:"present if parameter type is POSITIONAL"`
	BodyText              [][]string                        `json:"body_text,omitempty" description:"present if parameter type is POSITIONAL"`
	HeaderLocation        *SendTemplateParameterLocation    `json:"header_location,omitempty"`
}

type TemplateComponentTextNamedParam struct {
	Example   string `json:"example" val:"required" description:"sample text"`
	ParamName string `json:"param_name" val:"required" description:"name of the variable"`
}

type TemplateComponentButton struct {
	Type        types.WATemplateButtonType `json:"type" val:"required" description:"type of the button"`
	Text        string                     `json:"text" val:"required" description:"text of the button"`
	PhoneNumber string                     `json:"phone_number,omitempty" description:"present if type is PHONE_NUMBER"`
	Url         string                     `json:"url,omitempty" description:"present if type is URL"`
	TTLMinutes  int                        `json:"ttl_minutes,omitempty" description:"present if type is VOICE_CALL, max 30 days"`
	Example     []string                   `json:"example,omitempty" description:"for preview"`
	// flow
	// FlowAction     types.WATemplateFlowAction `json:"flow_action,omitempty" description:"present if type is FLOW"`
	// FlowId         string                     `json:"flow_id,omitempty" description:"present if type is FLOW"`
	// NavigateScreen string                     `json:"navigate_screen" description:"present if type is FLOW"`
}

func classifyDocumentType(documentURL string) string {
	lowerURL := strings.ToLower(documentURL)
	switch {
	case strings.Contains(lowerURL, ".pdf"):
		return "pdf"
	case strings.Contains(lowerURL, ".docx"), strings.Contains(lowerURL, ".doc"):
		return "doc"
	case strings.Contains(lowerURL, ".xlsx"), strings.Contains(lowerURL, ".xls"), strings.Contains(lowerURL, ".csv"):
		return "sheet"
	case strings.Contains(lowerURL, ".pptx"), strings.Contains(lowerURL, ".ppt"):
		return "ppt"
	case strings.Contains(lowerURL, ".txt"):
		return "txt"
	case strings.Contains(lowerURL, ".zip"):
		return "zip"
	default:
		return "file"
	}
}

func documentIconForType(documentType string) string {
	switch documentType {
	case "pdf":
		return "📄"
	case "doc":
		return "📝"
	case "sheet":
		return "📊"
	case "ppt":
		return "📽️"
	case "txt":
		return "📃"
	case "zip":
		return "🗜️"
	default:
		return "📁"
	}
}

func formatWhatsAppText(text string) string {
	var formatted strings.Builder
	for len(text) > 0 {
		if strings.HasPrefix(text, "{{") {
			if closingIndex := strings.Index(text[2:], "}}"); closingIndex >= 0 {
				contentEnd := closingIndex + 2
				formatted.WriteString(html.EscapeString(text[:contentEnd+2]))
				text = text[contentEnd+2:]
				continue
			}
		}

		if strings.HasPrefix(text, "```") {
			if closingIndex := indexOutsideTemplatePlaceholder(text, "```", 3); closingIndex >= 0 {
				formatted.WriteString("<span style='font-family:monospace;background:#f0f2f5;padding:1px 3px;border-radius:3px;'>")
				formatted.WriteString(html.EscapeString(text[3:closingIndex]))
				formatted.WriteString("</span>")
				text = text[closingIndex+3:]
				continue
			}
		}

		if text[0] == '*' || text[0] == '_' || text[0] == '~' {
			marker := text[0]
			if closingIndex := indexOutsideTemplatePlaceholder(text, string(marker), 1); closingIndex > 1 {
				openingTag, closingTag := "", ""
				switch marker {
				case '*':
					openingTag, closingTag = "<strong style='font-weight:700;'>", "</strong>"
				case '_':
					openingTag, closingTag = "<em style='font-style:italic;'>", "</em>"
				case '~':
					openingTag, closingTag = "<del style='text-decoration:line-through;'>", "</del>"
				}
				formatted.WriteString(openingTag)
				formatted.WriteString(html.EscapeString(text[1:closingIndex]))
				formatted.WriteString(closingTag)
				text = text[closingIndex+1:]
				continue
			}
		}
		nextSpecial := strings.IndexAny(text, "*_~`")
		if placeholderIndex := strings.Index(text, "{{"); placeholderIndex >= 0 && (nextSpecial < 0 || placeholderIndex < nextSpecial) {
			nextSpecial = placeholderIndex
		}
		if nextSpecial <= 0 {
			if nextSpecial == 0 {
				nextSpecial = 1
			} else {
				nextSpecial = len(text)
			}
		}
		formatted.WriteString(html.EscapeString(text[:nextSpecial]))
		text = text[nextSpecial:]
	}
	return formatted.String()
}

func indexOutsideTemplatePlaceholder(text string, marker string, start int) int {
	for index := start; index < len(text); {
		if strings.HasPrefix(text[index:], "{{") {
			closingIndex := strings.Index(text[index+2:], "}}")
			if closingIndex >= 0 {
				index += closingIndex + 4
				continue
			}
		}
		if strings.HasPrefix(text[index:], marker) {
			return index
		}
		index++
	}
	return -1
}

// #endregion

// #region for front end to know how to send

type SendTemplate struct {
	Components []SendTemplateComponent `json:"components"`
}

// to payload to be sent to Meta, use concrete SendTemplate
func (sendTemplate *SendTemplate) FinalPayload(template *Template, customer dao_customer.Customer, broadcastToken string) (map[string]any, error) {
	// load customer data
	for _, component := range sendTemplate.Components {
		for i, parameter := range component.Parameters {
			switch parameter.Source {
			case "customer.name":
				component.Parameters[i].Text = customer.DisplayName
			case "customer.token":
				component.Parameters[i].Text = customer.Token
			}
			if parameter.Type == "coupon_code" { // should do at front end, but just correct it here
				if component.Parameters[i].CouponCode == "" {
					component.Parameters[i].CouponCode = component.Parameters[i].Text
				}
				component.Parameters[i].Text = ""
			}
		}
	}
	// add additional buttons for flow and quick_reply
	var additionalButtons []SendTemplateComponent
	for _, component := range template.Components {
		for i, button := range component.Buttons {
			// both these 2 are not added to SendTemplate, so we can safely add them here
			parameter := SendTemplateParameter{}
			additionalButton := SendTemplateComponent{
				Type: "button",
			}
			switch button.Type {
			case "FLOW":
				parameter.Type = "action"
				parameter.Action = &SendTemplateParameterAction{
					FlowToken: broadcastToken,
				}
				additionalButton.Type = "button"
				additionalButton.SubType = "flow"
				additionalButton.Index = fmt.Sprint(i)
				additionalButton.Parameters = []SendTemplateParameter{
					parameter,
				}
				additionalButtons = append(additionalButtons, additionalButton)
			case "QUICK_REPLY":
				parameter.Type = "payload"
				parameter.Payload = broadcastToken
				additionalButton.Type = "button"
				additionalButton.SubType = "quick_reply"
				additionalButton.Index = fmt.Sprint(i)
				additionalButton.Parameters = []SendTemplateParameter{
					parameter,
				}
				additionalButtons = append(additionalButtons, additionalButton)
			}
		}
	}
	if len(additionalButtons) > 0 {
		sendTemplate.Components = append(sendTemplate.Components, additionalButtons...)
	}
	// clean up any component with no parameter
	componentsWithParameters := make([]SendTemplateComponent, 0, len(sendTemplate.Components))
	for _, component := range sendTemplate.Components {
		if len(component.Parameters) > 0 {
			componentsWithParameters = append(componentsWithParameters, component)
		}
	}
	sendTemplate.Components = componentsWithParameters
	// reset illegal parameters
	for i, component := range sendTemplate.Components {
		for ii := range component.Parameters {
			sendTemplate.Components[i].Parameters[ii].InputIndex = 0
			sendTemplate.Components[i].Parameters[ii].InputRequired = false
			sendTemplate.Components[i].Parameters[ii].InputMaxLength = 0
			sendTemplate.Components[i].Parameters[ii].InputTitle = ""
			sendTemplate.Components[i].Parameters[ii].Source = ""
		}
	}
	// payload := map[string]any{
	// 	"messaging_product": "whatsapp",
	// 	"to":                customer.WAId,
	// 	"type":              "template",
	// 	"template": map[string]any{
	// 		"name": template.Name,
	// 		"language": map[string]string{
	// 			"code": template.Language,
	// 		},
	// 		"components": sendTemplate.Components,
	// 	},
	// }
	// createMessage feature will supply the rest
	payload := map[string]any{
		"name": template.Name,
		"language": map[string]string{
			"code": template.Language,
		},
		"components": sendTemplate.Components,
	}
	if cfg.Default().Site.Environment == types.EnvironmentDevelop {
		sendTemplateData, err := json.Marshal(sendTemplate)
		if err != nil {
			return nil, err
		}
		helper.WriteToFile(string(sendTemplateData), "files/wa/send_template_payload.json")
	}
	return payload, nil
}

type SendTemplateComponent struct {
	Type       string                  `json:"type"`               // header, body, button
	SubType    string                  `json:"sub_type,omitempty"` // only if type is button; url, quick_reply, copy_code, flow
	Index      string                  `json:"index,omitempty"`    // only if type is button; zero-based position of the button "0", "1"
	Parameters []SendTemplateParameter `json:"parameters,omitempty"`
}

type SendTemplateParameter struct {
	Type string `json:"type"` // text, image, video, document, location, payload, coupon_code, action
	// component.type = header, body, button, they are the variables not full text
	ParameterName string `json:"parameter_name,omitempty"` // only for text
	Text          string `json:"text,omitempty"`           // only for text
	Source        string `json:"source,omitempty"`         // broadcast-only; strip before Meta, if it's dynamic like: customer.name, customer.additional_data.birthday; will be computed and put in text
	// component.type = header, audio is not supported
	Image    *SendTemplateMedia             `json:"image,omitempty"`    // type = image
	Video    *SendTemplateMedia             `json:"video,omitempty"`    // type = video
	Document *SendTemplateParameterDocument `json:"document,omitempty"` // type = document
	Location *SendTemplateParameterLocation `json:"location,omitempty"` // type = location
	// component.type = body
	Currency *SendTemplateParameterCurrency `json:"currency,omitempty"`  // type = currency
	DateTime *SendTemplateParameterDateTime `json:"date_time,omitempty"` // type = date_time
	// component.type = button
	Payload    string                       `json:"payload,omitempty"`     // component.sub_type = quick_reply, type = payload
	CouponCode string                       `json:"coupon_code,omitempty"` // component.sub_type = copy_code, type = coupon_code
	Action     *SendTemplateParameterAction `json:"action,omitempty"`      // component.sub_type = flow, type = "action"
	// let user see, omitempty becuase it will be compiled to final payload to Meta
	InputIndex     int    `json:"input_index,omitempty"`      // one-based variable position in this component
	InputTitle     string `json:"input_title,omitempty"`      // Coupon Code, Quick Reply Payload etc...
	InputRequired  bool   `json:"input_required,omitempty"`   // if an input is required from user
	InputMaxLength int    `json:"input_max_length,omitempty"` // limit the length, e.g. coupon code
}

type SendTemplateMedia struct {
	Link string `json:"link,omitempty"` // if it's an url, only 1 is needed
	Id   string `json:"id,omitempty"`   // Meta media ID
}

type SendTemplateParameterDocument struct {
	SendTemplateMedia
	FileName string `json:"filename"`
}

type SendTemplateParameterLocation struct {
	Latitude  float32 `json:"latitude"`
	Longitude float32 `json:"longitude"`
	Name      string  `json:"name"`
	Address   string  `json:"address"`
}

type SendTemplateParameterCurrency struct {
	FallbackValue string `json:"fallback_value" example:"S$25.00"`
	Code          string `json:"code" example:"SGD"`
	Amount1000    int64  `json:"amount_1000" example:"25000"`
}

type SendTemplateParameterDateTime struct {
	FallbackValue string `json:"fallback_value" example:"9 September 2026, 10:30 AM"`
	Year          int64  `json:"year" example:"2026"`
	Month         int64  `json:"month" example:"9"`
	DayOfMonth    int64  `json:"day_of_month" example:"10"`
	Hour          int64  `json:"hour" example:"10"`
	Minute        int64  `json:"minute" example:"30"`
}

type SendTemplateParameterAction struct {
	FlowToken      string         `json:"flow_token"`
	FlowActionData map[string]any `json:"flow_action_data,omitempty"`
}

func (template *Template) GetSendComponents() []SendTemplateComponent {
	components := make([]SendTemplateComponent, 0)
	for _, component := range template.Components {
		switch component.Type {
		case types.WATemplateComponentTypeHeader:
			parameters := []SendTemplateParameter{}
			switch component.Format {
			case types.WATemplateComponentFormatText:
				parameters = templateTextSendParameters(component.Text, template.ParameterFormat)
			case types.WATemplateComponentFormatImage, types.WATemplateComponentFormatVideo, types.WATemplateComponentFormatDocument, types.WATemplateComponentFormatLocation:
				parameters = append(parameters, SendTemplateParameter{Type: strings.ToLower(string(component.Format))})
			}
			if len(parameters) > 0 {
				components = append(components, SendTemplateComponent{Type: "header", Parameters: parameters})
			}
		case types.WATemplateComponentTypeBody:
			if parameters := templateTextSendParameters(component.Text, template.ParameterFormat); len(parameters) > 0 {
				components = append(components, SendTemplateComponent{Type: "body", Parameters: parameters})
			}
		case types.WATemplateComponentTypeButtons:
			for index, button := range component.Buttons {
				parameter := SendTemplateParameter{}
				switch button.Type {
				case types.WATemplateButtonTypeURL:
					if !strings.Contains(button.Url, "{{") {
						continue
					}
					parameter.Type = "text"
					parameter.InputRequired = true
					parameter.InputTitle = "URL"
				case types.WATemplateButtonTypeCopyCode:
					parameter.Type = "coupon_code"
					parameter.InputTitle = "Conpon Code"
					parameter.InputRequired = true
					parameter.InputMaxLength = 20
				default:
					continue
				}
				components = append(components, SendTemplateComponent{
					Type:       "button",
					SubType:    strings.ToLower(string(button.Type)),
					Index:      fmt.Sprint(index),
					Parameters: []SendTemplateParameter{parameter},
				})
			}
		}
	}
	return components
}

func templateTextSendParameters(text string, parameterFormat types.WATemplateParameterFormat) []SendTemplateParameter {
	parameters := []SendTemplateParameter{}
	for remainingText := text; ; {
		start := strings.Index(remainingText, "{{")
		if start < 0 {
			break
		}
		end := strings.Index(remainingText[start+2:], "}}")
		if end < 0 {
			break
		}
		parameterName := strings.TrimSpace(remainingText[start+2 : start+2+end])
		if parameterName != "" {
			inputIndex := len(parameters) + 1
			var parameter SendTemplateParameter
			if parameterFormat == types.WATemplateParameterFormatNamed {
				parameter = SendTemplateParameter{
					ParameterName: parameterName,
					InputTitle:    fmt.Sprintf("{{%s}}", parameterName),
				}
			} else {
				parameter = SendTemplateParameter{
					InputIndex: inputIndex,
					InputTitle: fmt.Sprintf("{{%d}}", inputIndex),
				}
			}
			parameter.Type = "text"
			parameter.InputRequired = true
			parameters = append(parameters, parameter)
		}
		remainingText = remainingText[start+2+end+2:]
	}
	return parameters
}

// # endregion
