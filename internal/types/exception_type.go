package types

type ExceptionType string

const (
	ExceptionTypeDatabase            ExceptionType = "DATABAE"
	ExceptionTypeUnauthorized        ExceptionType = "UNAUTHORIZED"
	ExceptionTypePageNotFound        ExceptionType = "PAGE_NOT_FOUND"
	ExceptionEntityTypeNotFound      ExceptionType = "ENTITY_NOT_FOUND"
	ExceptionTypeMethodNotAllowed    ExceptionType = "METHOD_NOT_ALLOWED"
	ExceptionTypeJSON                ExceptionType = "JSON"
	ExceptionTypeQuery               ExceptionType = "QUERY"
	ExceptionTypeURIParser           ExceptionType = "URI"
	ExceptionTypeHeaderParser        ExceptionType = "HEADER"
	ExceptionTypeInvalidInput        ExceptionType = "INVALID_INPUT"
	ExceptionTypeInternalServer      ExceptionType = "INTERNAL_SERVER_ERROR"
	ExceptionTypeInapproriateContent ExceptionType = "INAPPRORIATE_CONTENT"
	ExceptionTypeTypeOther           ExceptionType = "OTHER"
	ExceptionTypeMissingAPIKey       ExceptionType = "MISSING_API_KEY"
	ExceptionTypeTooManyRequest      ExceptionType = "TOO_MANY_REQUESTS"
)

const (
	ExceptionMessageInternalServerError string = "Sorry, something is wrong with our server."
	ExceptionMessageBadGateway          string = "Sorry, an error occured at the vendor."
)
