package dto_wa

type Incoming struct {
	Object string          `json:"object"`
	Entry  []IncomingEntry `json:"entry"`
}

type IncomingEntry struct {
	ID      string           `json:"id"`
	Changes []IncomingChange `json:"changes"`
}

type IncomingChange struct {
	Field string        `json:"field"`
	Value IncomingValue `json:"value"`
}

type IncomingValue struct {
	MessagingProduct string            `json:"messaging_product"`
	Metadata         IncomingMetadata  `json:"metadata"`
	Contacts         []IncomingContact `json:"contacts,omitempty"`
	Messages         []IncomingMessage `json:"messages,omitempty"`
	Statuses         []Status          `json:"statuses,omitempty"`
	SyncStatus       string            `json:"sync_status,omitempty"`
	ChunkNumber      int               `json:"chunk_number,omitempty"`
}

type IncomingMetadata struct {
	DisplayPhoneNumber string `json:"display_phone_number"`
	PhoneNumberID      string `json:"phone_number_id"`
}

type IncomingContact struct {
	Profile IncomingProfile `json:"profile"`
	WaID    string          `json:"wa_id"`
}

type IncomingProfile struct {
	Name string `json:"name"`
}

type IncomingMessage struct {
	PhoneNumberID string              `json:"phone_number_id,omitempty"`
	From          string              `json:"from"`
	ID            string              `json:"id"`
	Text          IncomingMessageText `json:"text,omitempty"`
	Timestamp     string              `json:"timestamp"`
	Type          string              `json:"type"`
}

type IncomingMessageText struct {
	Body string `json:"body"`
}

type Status struct {
	ID           string              `json:"id"`
	Status       string              `json:"status"`
	Timestamp    string              `json:"timestamp"`
	RecipientID  string              `json:"recipient_id"`
	Conversation *StatusConversation `json:"conversation,omitempty"`
	Pricing      *StatusPricing      `json:"pricing,omitempty"`
	Errors       []StatusError       `json:"errors,omitempty"`
}

type StatusConversation struct {
	ID                  string                    `json:"id"`
	ExpirationTimestamp string                    `json:"expiration_timestamp,omitempty"`
	Origin              *StatusConversationOrigin `json:"origin,omitempty"`
}

type StatusConversationOrigin struct {
	Type string `json:"type,omitempty"`
}

type StatusPricing struct {
	Billable     bool   `json:"billable,omitempty"`
	PricingModel string `json:"pricing_model,omitempty"`
	Category     string `json:"category,omitempty"`
	Type         string `json:"type,omitempty"`
}

type StatusError struct {
	Code      int              `json:"code"`
	Title     string           `json:"title"`
	Message   string           `json:"message"`
	ErrorData *StatusErrorData `json:"error_data,omitempty"`
}

type StatusErrorData struct {
	Details string `json:"details"`
}
