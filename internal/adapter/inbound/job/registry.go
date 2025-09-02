package job

import jobport "github.com/jvdiamondtech/ms-notification-cat/internal/domain/ports/outbound/job"

// Registry Job註冊表
type Registry struct {
	jobs map[string]jobport.ScheduledJob
}

// NewRegistry 透過依賴注入初始化Job
func NewRegistry(
	campaignTrigger *MessageCampaignTriggerJob,
) *Registry {
	registry := &Registry{
		jobs: make(map[string]jobport.ScheduledJob),
	}

	// 要執行的Job在這裡註冊
	registry.Register(campaignTrigger)

	return registry
}

func (r *Registry) Register(jobs ...jobport.ScheduledJob) {
	for _, job := range jobs {
		r.jobs[job.GetName()] = job
	}
}

func (r *Registry) GetAllJobs() []jobport.ScheduledJob {
	jobs := make([]jobport.ScheduledJob, 0, len(r.jobs))
	for _, job := range r.jobs {
		jobs = append(jobs, job)
	}
	return jobs
}
