package helper

import (
	"fmt"
	"strings"

	"github.com/jjcheng/wawa-go/internal/types"
)

func NormalizeWAId(phoneNumber string) string {
	waId := strings.ReplaceAll(phoneNumber, "+", "")
	waId = strings.ReplaceAll(waId, " ", "")
	waId = strings.ReplaceAll(waId, "-", "")
	return waId
}

func GetChatChannelName(phoneNumberId string, customerWAId string, customerMetaUserId string) string {
	customerId := customerWAId
	if customerId == "" {
		customerId = customerMetaUserId
	}
	return fmt.Sprintf("chat:%s:%s", phoneNumberId, customerId)
}

func CanTransitionWAMessageStatus(current types.WAMessageStatus, next types.WAMessageStatus) bool {
	if current == next {
		return false
	}
	if next == types.WAMessageStatusFailed {
		switch current {
		case types.WAMessageStatusAccepted, types.WAMessageStatusHeldForQualityAssessment, types.WAMessageStatusPaused, types.WAMessageStatusSent:
			return true
		default:
			return false
		}
	}
	if current == types.WAMessageStatusFailed {
		return false
	}
	return waMessageStatusRank(next) > waMessageStatusRank(current)
}

func waMessageStatusRank(status types.WAMessageStatus) int {
	switch status {
	case types.WAMessageStatusAccepted, types.WAMessageStatusHeldForQualityAssessment, types.WAMessageStatusPaused:
		return 1
	case types.WAMessageStatusSent:
		return 2
	case types.WAMessageStatusDelivered:
		return 3
	case types.WAMessageStatusRead:
		return 4
	case types.WAMessageStatusPlayed:
		return 5
	default:
		return 0
	}
}
