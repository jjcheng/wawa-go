package exception

import (
	"net/http"

	"github.com/jjcheng/wawa-go/internal/types"
)

// this can be safely returned to the user as it does not contain sensitive error info
type Exception struct {
	Message    string `json:"message" description:"description of the error" example:"Internal server error"`
	StatusCode int    `json:"status_code" description:"http status code" example:"500"`
}

func NewCustomException(message string, statusCode int) *Exception {
	exception := Exception{Message: message, StatusCode: statusCode}
	return &exception
}

func NewException(exceptionType types.ExceptionType) *Exception {
	switch exceptionType {
	case types.ExceptionTypeJSON:
		return NewCustomException("invalid JSON", http.StatusBadRequest)
	case types.ExceptionTypeQuery:
		return NewCustomException("invalid query", http.StatusBadRequest)
	case types.ExceptionTypeURIParser:
		return NewCustomException("invalid URI", http.StatusBadRequest)
	case types.ExceptionTypeHeaderParser:
		return NewCustomException("invalid header", http.StatusBadRequest)
	case types.ExceptionTypeUnauthorized:
		return NewCustomException("unauthorized access", http.StatusUnauthorized)
	case types.ExceptionEntityTypeNotFound:
		return NewCustomException("resource not found", http.StatusNotFound)
	case types.ExceptionTypeDatabase:
		return NewCustomException("database error", http.StatusInternalServerError)
	case types.ExceptionTypePageNotFound:
		return NewCustomException("page not found", http.StatusNotFound)
	case types.ExceptionTypeMethodNotAllowed:
		return NewCustomException("invalid method", http.StatusMethodNotAllowed)
	case types.ExceptionTypeInvalidInput:
		return NewCustomException("invalid input", http.StatusBadRequest)
	case types.ExceptionTypeInapproriateContent:
		return NewCustomException("inappropriate input content", http.StatusBadRequest)
	case types.ExceptionTypeMissingAPIKey:
		return NewCustomException("missing api key in header", http.StatusBadRequest)
	case types.ExceptionTypeTooManyRequest:
		return NewCustomException("too many requests", http.StatusTooManyRequests)
	default:
		return NewCustomException("internal server error", http.StatusInternalServerError)
	}
}
