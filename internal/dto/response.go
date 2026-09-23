package dto

import (
	"net/http"
	"time"

	"github.com/jjcheng/wawa-go/internal/exception"
)

type Response[T any] struct {
	ResponseBase
	Error error `json:"-"`
	Data  T     `json:"data,omitempty" description:"the actual API returned data, empty if not success"`
}

type ResponseBase struct {
	Success     bool                       `json:"success" val:"required" description:"is request is success" example:"true"`
	StatusCode  int                        `json:"status_code" val:"required" description:"http status code" example:"200"`
	Message     string                     `json:"message,omitempty" description:"error message if not success" example:"owner not found"`
	RequestId   string                     `json:"request_id,omitempty" description:"request correlation id"`
	StartAt     time.Time                  `json:"-"`
	EndAt       time.Time                  `json:"-"`
	TimeTaken   int64                      `json:"time_taken" val:"required" description:"time taken in miliseconds" example:"100"`
	InputErrors []exception.InputException `json:"input_errors,omitempty" description:"list of input violations with details if any"`
}

func NewSuccessResponse[T any](data T) Response[T] {
	return Response[T]{
		ResponseBase: ResponseBase{
			Success:    true,
			StatusCode: http.StatusOK,
		},
		Data: data,
	}
}

func NewSuccessResponseWithMessage[T any](data T, message string) Response[T] {
	return Response[T]{
		ResponseBase: ResponseBase{
			Success:    true,
			StatusCode: http.StatusOK,
			Message:    message,
		},
		Data: data,
	}
}

func NewFailedResponse[T any](statusCode int, message string, err error) Response[T] {
	return Response[T]{
		Error: err,
		ResponseBase: ResponseBase{
			Success:    false,
			StatusCode: statusCode,
			Message:    message,
		},
	}
}

func NewEmptyResponse(success bool, statusCode int) Response[any] {
	return Response[any]{
		ResponseBase: ResponseBase{
			Success:    success,
			StatusCode: statusCode,
		},
	}
}

func NewInvalidInputResponse[T any](inputExceptions []exception.InputException) Response[T] {
	return Response[T]{
		ResponseBase: ResponseBase{
			Success:     false,
			StatusCode:  http.StatusBadRequest,
			InputErrors: inputExceptions,
			Message:     *exception.GetInputExceptionsDescription(inputExceptions),
		},
	}
}
