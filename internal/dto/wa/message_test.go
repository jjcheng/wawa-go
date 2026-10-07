package dto_wa

import "testing"

func TestMessageText(t *testing.T) {
	tests := []struct {
		name    string
		message Message
		want    string
	}{
		{
			name: "text message",
			message: Message{
				Type:    "text",
				Payload: map[string]any{"text": map[string]any{"body": "hello"}},
			},
			want: "hello",
		},
		{
			name: "non-text message",
			message: Message{
				Type:    "image",
				Payload: map[string]any{"text": map[string]any{"body": "hello"}},
			},
		},
		{
			name: "missing text payload",
			message: Message{
				Type:    "text",
				Payload: map[string]any{},
			},
		},
		{
			name: "malformed text payload",
			message: Message{
				Type:    "text",
				Payload: map[string]any{"text": "hello"},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := test.message.Text(); got != test.want {
				t.Errorf("Text() = %q, want %q", got, test.want)
			}
		})
	}
}
