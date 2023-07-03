package orchestrator

import "github.com/samber/lo"

type MappedSchedulerProvider struct {
	schedulers        map[string]Scheduler
	enabledSchedulers []string
}

func NewMappedSchedulerProvider(schedulers map[string]Scheduler) *MappedSchedulerProvider {
	return &MappedSchedulerProvider{
		schedulers:        schedulers,
		enabledSchedulers: lo.Keys(schedulers),
	}
}

func (p *MappedSchedulerProvider) Scheduler(jobType string) Scheduler {
	return p.schedulers[jobType]
}

func (p *MappedSchedulerProvider) EnabledSchedulers() []string {
	return p.enabledSchedulers
}

// compile time check whether the MappedSchedulerProvider implements the SchedulerProvider interface
var _ SchedulerProvider = (*MappedSchedulerProvider)(nil)
