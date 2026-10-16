package dto_ai

type ChatResponse struct {
	Message  string
	NextStep *ChatNextStep
}

func NewChatResponse(message string, nextStep *ChatNextStep) ChatResponse {
	return ChatResponse{
		Message:  message,
		NextStep: nextStep,
	}
}
