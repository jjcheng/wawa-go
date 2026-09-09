package helper

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/jjcheng/wawa-go/internal/types"
	"github.com/nyaruka/phonenumbers"
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

// take X-Hub-Signature-256 from header
func VerifyWhatsAppWebhookSignature(signature string, body []byte, appSecret string) bool {
	signature = strings.TrimPrefix(strings.TrimSpace(signature), "sha256=")
	if signature == "" || strings.TrimSpace(appSecret) == "" {
		return false
	}
	providedSignature, err := hex.DecodeString(signature)
	if err != nil {
		return false
	}
	mac := hmac.New(sha256.New, []byte(appSecret))
	_, _ = mac.Write(body)
	return hmac.Equal(providedSignature, mac.Sum(nil))
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

// waid = 6590000000
func GetCountryCodeAndPhoneNumberFromWAId(waID string) (string, string, error) {
	phoneNumber, err := phonenumbers.Parse("+"+waID, "")
	if err != nil || !phonenumbers.IsValidNumber(phoneNumber) {
		return "", "", fmt.Errorf("invalid WhatsApp ID %q", waID)
	}
	return fmt.Sprintf("%d", phoneNumber.GetCountryCode()), phonenumbers.GetNationalSignificantNumber(phoneNumber), nil
}
