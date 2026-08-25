package types

type ChatModel string

const (
	ChatModelFast                  ChatModel = "FAST"
	ChatModelDefault               ChatModel = "DEFAULT"
	ChatModelDefaultReasoningLow   ChatModel = "DEFAULT_REASONING_LOW"
	ChatModelDefaultReasoningHigh  ChatModel = "DEFAULT_REASONING_HIGH"
	ChatModelAdvanced              ChatModel = "ADVANCED"
	ChatModelAdvancedReasoningLow  ChatModel = "ADVANCED_REASONING_LOW"
	ChatModelAdvancedReasoningHigh ChatModel = "ADVANCED_REASONING_HIGH"
	ChatModelCoder                 ChatModel = "CODER"
)

var ChatModelTypes = []ChatModel{
	ChatModelFast,
	ChatModelDefault,
	ChatModelDefaultReasoningLow,
	ChatModelDefaultReasoningHigh,
	ChatModelAdvanced,
	ChatModelAdvancedReasoningLow,
	ChatModelAdvancedReasoningHigh,
	ChatModelCoder,
}
