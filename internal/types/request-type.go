package types

type HttpRequestType string

const (
	HttpRequestTypeUri      HttpRequestType = "URI"
	HttpRequestTypeQuery    HttpRequestType = "QUERY"
	HttpRequestTypeUriQuery HttpRequestType = "URI_QUERY"
	HttpRequestTypeJSON     HttpRequestType = "JSON"
	HttpRequestTypeUriJSON  HttpRequestType = "URI_JSON"
)
