package dto_wa

import (
	"strings"
	"testing"

	"github.com/jjcheng/wawa-go/internal/types"
)

func TestFormatWhatsAppTextPreservesMarkersInsideTemplateVariables(t *testing.T) {
	input := "{{customer_name}} {{customer*name}} {{customer~name}} {{customer`name}}"
	if actual := formatWhatsAppText(input); actual != input {
		t.Fatalf("formatWhatsAppText() = %q, want %q", actual, input)
	}
}

func TestFormatWhatsAppTextDoesNotCloseFormattingInsideTemplateVariable(t *testing.T) {
	input := "_Hello {{customer_name}}_"
	expected := "<em style='font-style:italic;'>Hello {{customer_name}}</em>"
	if actual := formatWhatsAppText(input); actual != expected {
		t.Fatalf("formatWhatsAppText() = %q, want %q", actual, expected)
	}
}

func TestFormatWhatsAppTextStillFormatsItalics(t *testing.T) {
	expected := "Hello <em style='font-style:italic;'>customer</em>"
	if actual := formatWhatsAppText("Hello _customer_"); actual != expected {
		t.Fatalf("formatWhatsAppText() = %q, want %q", actual, expected)
	}
}

func TestApplySendTemplateUsesActualValuesInPreview(t *testing.T) {
	template := Template{TemplateBase: TemplateBase{
		ParameterFormat: types.WATemplateParameterFormatPositional,
		Components: []TemplateComponent{
			{Type: types.WATemplateComponentTypeHeader, Format: types.WATemplateComponentFormatImage, Example: &TemplateComponentExample{HeaderHandle: []string{"sample.jpg"}}},
			{Type: types.WATemplateComponentTypeBody, Format: types.WATemplateComponentFormatText, Text: "Hello {{1}}, total {{2}}", Example: &TemplateComponentExample{BodyText: [][]string{{"sample", "$0"}}}},
			{Type: types.WATemplateComponentTypeButtons, Buttons: []TemplateComponentButton{{Type: types.WATemplateButtonTypeURL, Text: "Open", Url: "https://example.com/{{1}}"}}},
		},
	}}
	sendTemplate := SendTemplate{Components: []SendTemplateComponent{
		{Type: "header", Parameters: []SendTemplateParameter{{Type: "image", Image: &SendTemplateMedia{Link: "https://example.com/actual.jpg"}}}},
		{Type: "body", Parameters: []SendTemplateParameter{{Type: "text", Text: "Alice"}, {Type: "currency", Currency: &SendTemplateParameterCurrency{FallbackValue: "S$25.00"}}}},
		{Type: "button", SubType: "url", Index: "0", Parameters: []SendTemplateParameter{{Type: "text", Text: "receipt-1"}}},
	}}

	template.ApplySendTemplate(sendTemplate)
	html := template.HTML(true, false)
	for _, expected := range []string{"actual.jpg", "Hello Alice, total S$25.00", "https://example.com/receipt-1"} {
		if !strings.Contains(html, expected) {
			t.Errorf("HTML() does not contain %q: %s", expected, html)
		}
	}
}

func TestApplySendTemplateUsesNamedTextValues(t *testing.T) {
	template := Template{TemplateBase: TemplateBase{
		Components: []TemplateComponent{{
			Type: types.WATemplateComponentTypeBody, Format: types.WATemplateComponentFormatText, Text: "Hello {{customer_name}}",
			Example: &TemplateComponentExample{BodyTextNamedParams: []TemplateComponentTextNamedParam{{ParamName: "customer_name", Example: "sample"}}},
		}},
	}}
	sendTemplate := SendTemplate{Components: []SendTemplateComponent{{
		Type: "body", Parameters: []SendTemplateParameter{{Type: "text", ParameterName: "customer_name", Text: "Alice"}},
	}}}

	template.ApplySendTemplate(sendTemplate)
	if html := template.HTML(true, false); !strings.Contains(html, "Hello Alice") {
		t.Errorf("HTML() does not contain the actual named value: %s", html)
	}
}
