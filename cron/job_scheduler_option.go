package cron

import "github.com/blend/go-sdk/logger"

// JobSchedulerOption is an option for job schedulers.
type JobSchedulerOption func(*JobScheduler)

// OptJobSchedulerTracer sets the job scheduler tracer.
func OptJobSchedulerTracer(tracer Tracer) JobSchedulerOption {
	return func(js *JobScheduler) { js.Tracer = tracer }
}

// OptJobSchedulerLog sets the job scheduler logger.
func OptJobSchedulerLog(log logger.FullReceiver) JobSchedulerOption {
	return func(js *JobScheduler) { js.Log = log }
}

// OptJobSchedulerHistoryConfig sets the job scheduler history config.
func OptJobSchedulerHistoryConfig(hc HistoryConfig) JobSchedulerOption {
	return func(js *JobScheduler) { js.HistoryConfig = hc }
}