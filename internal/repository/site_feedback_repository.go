package repository

import dao_site "github.com/jjcheng/wawa-go/internal/dao/site"

type SiteFeedbackRepository interface {
	Repository[dao_site.Feedback]
}
