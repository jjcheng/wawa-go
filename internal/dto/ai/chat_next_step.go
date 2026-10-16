package dto_ai

import "github.com/jjcheng/wawa-go/internal/types"

type ChatNextStep struct {
	Type  types.AIChatNextStepType
	Title string
	Body  string
}

func NewChatNextStep(typ types.AIChatNextStepType, title string, body string) ChatNextStep {
	return ChatNextStep{
		Type:  typ,
		Title: title,
		Body:  body,
	}
}
