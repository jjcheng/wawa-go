package dao_site

import "github.com/jjcheng/wawa-go/internal/dao"

type Feedback struct {
	dao.DAOBase
	UserId  int32  `json:"user_id"`
	Content string `json:"content"`
}

func (Feedback) TableName() string {
	return "site.feedbacks"
}

func (feedback Feedback) Base() dao.DAOBase {
	return feedback.DAOBase
}
