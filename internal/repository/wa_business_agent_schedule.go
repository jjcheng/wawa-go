package repository

import dao_wa "github.com/jjcheng/wawa-go/internal/dao/wa"

type WABusinessAgentScheduleRepository interface {
	Repository[dao_wa.BusinessAgentSchedule]
}
