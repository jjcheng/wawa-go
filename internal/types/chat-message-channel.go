package types

type ChatMessageChannel string

const (
	ChatMessageChannelWA  ChatMessageChannel = "WHATSAPP"
	ChatMessageChannelWeb ChatMessageChannel = "WEB"
)

var ChatMessageChannels = []ChatMessageChannel{
	ChatMessageChannelWA,
	ChatMessageChannelWeb,
}
