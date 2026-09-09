package dto_wa

import (
	"errors"
	"fmt"
	"html"
	"strings"

	"github.com/jjcheng/wawa-go/internal/exception"
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
}

type TemplateBase struct {
	Name            string                          `json:"name"`
	Category        types.WATemplateCategory        `json:"category"`
	Language        string                          `json:"language"`
	ParameterFormat types.WATemplateParameterFormat `json:"parameter_format,omitempty"`
	Components      []TemplateComponent             `json:"components,omitempty"`
	// lazy loaded
	PreviewHTML     string `json:"preview_html,omitempty"`
	RawHTML         string `json:"raw_html,omitempty"`
	PreviewDarkHTML string `json:"preview_dark_html,omitempty"`
	RawDarkHTML     string `json:"raw_dark_html,omitempty"`
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
	return fmt.Sprintf("<div style='box-sizing:border-box;width:100%%;max-width:360px;padding:12px 12px 20px;background:%s;font-family:Arial,sans-serif;'><div style='position:relative;overflow:hidden;width:100%%;max-width:330px;background:%s;border:1px solid %s;border-radius:8px;color:%s;'>%s</div></div>", theme.CanvasBackground, theme.CardBackground, theme.CardBorder, theme.TextPrimary, html.String())
}

// #end region

// #region list

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

// #endregion

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
	TemplateID string `json:"template_id"`
	Start      int64  `json:"start"`
	End        int64  `json:"end"`
	Sent       int64  `json:"sent,omitempty"`
	Delivered  int64  `json:"delivered,omitempty"`
	Read       int64  `json:"read,omitempty"`
	Clicked    int64  `json:"clicked,omitempty"`
}

type TemplateAnalyticsListResponse struct {
	Data   []TemplateAnalytics `json:"data"`
	Paging *TemplatePaging     `json:"paging,omitempty"`
}

// #endregion

// #region create

type TemplateComponent struct {
	Format  types.WATemplateComponentFormat `json:"format" val:"required" description:"TEXT, IMAGE, VIDEO, DOCUMENT, LOCATION"`
	Text    string                          `json:"text" description:"only if format = TEXT"`
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
		if strings.HasPrefix(text, "```") {
			if closingIndex := strings.Index(text[3:], "```"); closingIndex >= 0 {
				contentEnd := closingIndex + 3
				formatted.WriteString("<span style='font-family:monospace;background:#f0f2f5;padding:1px 3px;border-radius:3px;'>")
				formatted.WriteString(html.EscapeString(text[3:contentEnd]))
				formatted.WriteString("</span>")
				text = text[contentEnd+3:]
				continue
			}
		}

		if text[0] == '*' || text[0] == '_' || text[0] == '~' {
			marker := text[0]
			if closingIndex := strings.IndexByte(text[1:], marker); closingIndex > 0 {
				contentEnd := closingIndex + 1
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
				formatted.WriteString(html.EscapeString(text[1:contentEnd]))
				formatted.WriteString(closingTag)
				text = text[contentEnd+1:]
				continue
			}
		}

		nextMarker := strings.IndexAny(text, "*_~`")
		if nextMarker <= 0 {
			if nextMarker == 0 {
				nextMarker = 1
			} else {
				nextMarker = len(text)
			}
		}
		formatted.WriteString(html.EscapeString(text[:nextMarker]))
		text = text[nextMarker:]
	}
	return formatted.String()
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
			return fmt.Sprintf("<div style='margin:0;overflow:hidden;border-radius:6px;line-height:0;'><img src='%s' alt='Template header' style='display:block;width:100%%;height:auto;max-height:260px;object-fit:cover;'></div>", templateComponent.Example.HeaderHandle[0]), nil
		case types.WATemplateComponentFormatVideo:
			if templateComponent.Example == nil || len(templateComponent.Example.HeaderHandle) == 0 {
				return "", errors.New("no example or no header_handle")
			}
			return fmt.Sprintf("<div style='padding:0;'><video controls preload='metadata' playsinline src='%s' style='display:block;width:100%%;max-height:260px;object-fit:cover;border-radius:6px;background:#0b141a;'></video></div>", templateComponent.Example.HeaderHandle[0]), nil
		case types.WATemplateComponentFormatDocument:
			if templateComponent.Example == nil || len(templateComponent.Example.HeaderHandle) == 0 {
				return "", errors.New("no example or no header_handle")
			}
			documentURL := templateComponent.Example.HeaderHandle[0]
			if strings.HasSuffix(strings.ToLower(documentURL), ".pdf") {
				return fmt.Sprintf("<div style='padding:0;'><iframe src='%s' title='PDF preview' loading='lazy' style='display:block;width:100%%;height:220px;border:0;border-radius:6px;background:%s;'></iframe><a href='%s' target='_blank' rel='noopener noreferrer' style='display:flex;align-items:center;gap:8px;margin-top:2px;padding:9px 10px;border-radius:4px;background:%s;color:%s;text-decoration:none;font-size:13px;'><span style='font-size:20px;'>📄</span><span>Open PDF</span></a></div>", documentURL, theme.MutedBackground, documentURL, theme.MutedBackground, theme.TextPrimary), nil
			}
			documentType := classifyDocumentType(documentURL)
			documentIcon := documentIconForType(documentType)
			return fmt.Sprintf("<div style='padding:0;'><a href='%s' target='_blank' rel='noopener noreferrer' style='display:flex;align-items:center;gap:10px;padding:12px;border-radius:6px;background:%s;color:%s;text-decoration:none;font-size:13px;'><span style='font-size:26px;line-height:1;'>%s</span><span style='overflow:hidden;text-overflow:ellipsis;white-space:nowrap;'>%s document</span></a></div>", documentURL, theme.MutedBackground, theme.TextPrimary, documentIcon, strings.ToUpper(documentType)), nil
		default:
			return fmt.Sprintf("<div style='padding:0;'><img src='%s' alt='Location map' style='display:block;width:100%%;height:170px;object-fit:cover;border-radius:6px 6px 0 0;'><div style='padding:8px 9px;background:%s;border-radius:0 0 6px 6px;'><div style='font-weight:600;font-size:13.5px;line-height:18px;color:%s;'>%s</div><div style='margin-top:2px;font-size:12px;line-height:16px;color:%s;'>%s</div></div></div>", "https://www.onemap.gov.sg/api/staticmap/getStaticImage?layerchosen=default&zoom=15&height=450&width=450&lat=1.3521&lng=103.844", theme.MutedBackground, theme.TextPrimary, "Location Name", theme.TextSecondary, "Location Address"), nil
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
