package types

type ChatMessageRole string

const (
	ChatMessageRoleUser            ChatMessageRole = "USER"
	ChatMessageRoleAssistant       ChatMessageRole = "ASSISTANT"
	ChatMessageRoleRewritten       ChatMessageRole = "REWRITTEN"
	ChatMessageRoleUserComplete    ChatMessageRole = "USER_COMPLETE" // contextual complete user message
	ChatMessageRoleUserRaw         ChatMessageRole = "USER_RAW"
	ChatMessageRoleIntentLabel     ChatMessageRole = "INTENT_LABEL"
	ChatMessageRoleEntities        ChatMessageRole = "ENTITIES"
	ChatMessageRoleCondensed       ChatMessageRole = "CONDENSED"
	ChatMessageRoleUnprocessed     ChatMessageRole = "UNPROCESSED"
	ChatMessageRoleRequestResponse ChatMessageRole = "REQUEST_RESPONSE"
	ChatMessageRoleSaved           ChatMessageRole = "SAVED"
	ChatMessageRoleChartJSON       ChatMessageRole = "CHART_JSON"
	//ChatMessageRoleUserTmp         ChatMessageRole = "USER_TMP"      // temporary, used in analyze add project name
	//ChatMessageRoleAssistantTmp    ChatMessageRole = "ASSISTANT_TMP" // temporary
	ChatMessageRolePresentable     ChatMessageRole = "PRESENTABLE" // data generated in analyzing, will be used as
	ChatMessageRoleHook            ChatMessageRole = "HOOK"
	ChatMessageRoleThinkingProcess ChatMessageRole = "THINKING_PROCESS"
	ChatMessageRoleAppointment     ChatMessageRole = "APPOINTMENT"
	ChatMessageRoleShortlist       ChatMessageRole = "SHORTLIST"
	ChatMessageRoleAlerts          ChatMessageRole = "ALERTS"
)

var ChatMessageRoles = []ChatMessageRole{
	ChatMessageRoleUser,
	ChatMessageRoleUserComplete,
	ChatMessageRoleUserRaw,
	ChatMessageRoleAssistant,
	ChatMessageRoleRewritten,
	ChatMessageRoleIntentLabel,
	ChatMessageRoleEntities,
	ChatMessageRoleCondensed,
	ChatMessageRoleRequestResponse,
	ChatMessageRoleUnprocessed,
	ChatMessageRoleSaved,
	ChatMessageRoleChartJSON,
	ChatMessageRolePresentable,
	//ChatMessageRoleUserTmp,
	//ChatMessageRoleAssistantTmp,
	ChatMessageRoleThinkingProcess,
	ChatMessageRoleAppointment,
	ChatMessageRoleShortlist,
	ChatMessageRoleAlerts,
}

var DisplayChatMessageRoles = []ChatMessageRole{
	ChatMessageRoleAssistant,
	ChatMessageRoleUser,
}
