package job

import "github.com/jvdiamondtech/ms-notification-cat/internal/domain/jobport"

// Registry Job註冊表
type Registry struct {
	jobs map[string]jobport.ScheduledJob
}

func NewRegistry(
	campaignTrigger jobport.ScheduledJob,
) *Registry {
	registry := &Registry{
		jobs: make(map[string]jobport.ScheduledJob),
	}

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
