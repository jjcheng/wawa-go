package helper

import "strings"

func NormalizeWAId(phoneNumber string) string {
	waId := strings.ReplaceAll(phoneNumber, "+", "")
	waId = strings.ReplaceAll(waId, " ", "")
	waId = strings.ReplaceAll(waId, "-", "")
	return waId
}
