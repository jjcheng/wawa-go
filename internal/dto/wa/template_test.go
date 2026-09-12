package dto_wa

import "testing"

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
