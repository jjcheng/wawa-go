package helper

import (
	"fmt"
	"strings"
)

func GetWebhsiteFullUrl(domainName string, commerceWebsiteMainDomain string) string {
	if strings.HasPrefix(domainName, "https://") {
		return domainName
	}
	return fmt.Sprintf("https://%s.%s", domainName, commerceWebsiteMainDomain)
}
