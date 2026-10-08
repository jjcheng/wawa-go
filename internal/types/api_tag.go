package types

type APITag string

const (
	APITagAdmin         APITag = "Admin"
	APITagUser          APITag = "User"
	APITagNotification  APITag = "Notification"
	APITagAuth          APITag = "Auth"
	APITagCustomer      APITag = "Customer"
	APITagBroadcast     APITag = "Broadcast"
	APITagWA            APITag = "WA"
	APITagBusinessAgent APITag = "BusinessAgent"
	APITagCommerce      APITag = "Commerce"
	APITagPublic        APITag = "Public"
	APITagSite          APITag = "Site"
	APITagAIWorker      APITag = "AIWorker"
	APITagAIAgent       APITag = "AIAgent"
)
