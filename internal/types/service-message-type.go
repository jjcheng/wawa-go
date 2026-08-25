package types

type ServiceMessageType string

const (
	ServiceMessageTypeReceiveMessage ServiceMessageType = "RECEIVE_MESSAGE"
	ServiceMessageTypePush           ServiceMessageType = "PUSH"
)
