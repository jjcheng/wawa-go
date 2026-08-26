package exception

import (
	"strings"

	"github.com/jjcheng/wawa-go/internal/helper"
)

type InputException struct {
	Field   string `json:"field" val:"required" description:"name of the input field" example:"name of the input field"`
	Message string `json:"message" val:"required" description:"description of the error" example:"description of the error"`
}

func NewInputException(field string, message string) InputException {
	inputException := InputException{Field: field, Message: message}
	return inputException
}

func GetInputExceptionsDescription(inputExceptions []InputException) *string {
	if len(inputExceptions) == 0 {
		return nil
	}
	text := []string{}
	for _, ex := range inputExceptions {
		text = append(text, ex.Message)
	}
	return helper.ConvertToPointer(strings.Join(text, "\n"))
}
