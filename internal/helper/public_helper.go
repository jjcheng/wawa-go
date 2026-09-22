package helper

import "fmt"

func GetWebsiteLastWhatsAppContactUserIdCacheKey(websiteId int32) string {
	return fmt.Sprintf("website_%d_last_whatsapp_user_id", websiteId)
}
