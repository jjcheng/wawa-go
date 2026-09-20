package helper

import "fmt"

func GetNotificationChannelName(userId int32) string {
	return fmt.Sprintf("notification:%d", userId)
}
