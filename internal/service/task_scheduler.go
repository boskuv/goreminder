package service

import (
	"context"
	"time"

	"github.com/go-co-op/gocron"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/boskuv/goreminder/internal/repository"
	"github.com/boskuv/goreminder/pkg/logger"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
)

// TaskScheduler handles scheduled task rescheduling operations
type TaskScheduler struct {
	taskRepo  repository.TaskRepository
	taskSvc   *TaskService
	tracer    trace.Tracer
	logger    zerolog.Logger
	scheduler *gocron.Scheduler
}

// NewTaskScheduler creates a new TaskScheduler
func NewTaskScheduler(taskRepo repository.TaskRepository, taskSvc *TaskService, logger zerolog.Logger) *TaskScheduler {
	// Create scheduler with UTC timezone
	s := gocron.NewScheduler(time.UTC)

	return &TaskScheduler{
		taskRepo:  taskRepo,
		taskSvc:   taskSvc,
		tracer:    otel.Tracer("task-scheduler"),
		logger:    logger,
		scheduler: s,
	}
}

// RunScheduledRescheduling runs the rescheduling check at specified times
// It runs twice a day at 00:00 and 12:00 UTC
func (s *TaskScheduler) RunScheduledRescheduling(ctx context.Context) error {
	ctx, span := s.tracer.Start(ctx, "task_scheduler.RunScheduledRescheduling")
	defer span.End()

	log := logger.WithTraceContext(ctx, s.logger)
	log.Info().
		Msg("starting scheduled task rescheduling check")

	// Get tasks that need rescheduling
	tasks, err := s.taskRepo.GetTasksNeedingRescheduling(ctx)
	if err != nil {
		log.Error().
			Stack().
			Err(err).
			Msg("failed to get tasks needing rescheduling")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return errors.WithStack(err)
	}

	if len(tasks) == 0 {
		log.Info().
			Msg("no tasks found needing rescheduling")
		span.SetAttributes(attribute.Int("tasks.count", 0))
		span.SetStatus(codes.Ok, "no tasks to reschedule")
		return nil
	}

	log.Info().
		Int("tasks.count", len(tasks)).
		Msg("found tasks needing rescheduling")

	// Reschedule all tasks
	if err := s.taskSvc.RescheduleTasks(ctx, tasks); err != nil {
		log.Error().
			Stack().
			Err(err).
			Int("tasks.count", len(tasks)).
			Msg("failed to reschedule tasks")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return errors.WithStack(err)
	}

	// Get tasks with cron expression and requires_confirmation = false that need rescheduling
	cronTasks, err := s.taskRepo.GetTasksWithCronNeedingRescheduling(ctx)
	if err != nil {
		log.Error().
			Stack().
			Err(err).
			Msg("failed to get tasks with cron needing rescheduling")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return errors.WithStack(err)
	}

	if len(cronTasks) > 0 {
		log.Info().
			Int("cron_tasks.count", len(cronTasks)).
			Msg("found cron tasks needing rescheduling")

		// Reschedule cron tasks (update start_date only, no queue publishing)
		if err := s.taskSvc.RescheduleCronTasks(ctx, cronTasks); err != nil {
			log.Error().
				Stack().
				Err(err).
				Int("cron_tasks.count", len(cronTasks)).
				Msg("failed to reschedule cron tasks")
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return errors.WithStack(err)
		}

		log.Info().
			Int("cron_tasks.count", len(cronTasks)).
			Msg("cron tasks rescheduling completed successfully")
	}

	log.Info().
		Int("tasks.count", len(tasks)).
		Int("cron_tasks.count", len(cronTasks)).
		Msg("scheduled task rescheduling check completed successfully")
	span.SetAttributes(
		attribute.Int("tasks.count", len(tasks)),
		attribute.Int("cron_tasks.count", len(cronTasks)),
	)
	span.SetStatus(codes.Ok, "scheduled rescheduling completed successfully")
	return nil
}

// StartScheduler starts the scheduler that runs at 00:00 and 12:00 UTC twice daily
// It uses go-co-op/gocron for efficient scheduling and continues until the context is cancelled
func (s *TaskScheduler) StartScheduler(ctx context.Context) {
	log := s.logger.With().
		Str("component", "task_scheduler").
		Logger()

	log.Info().
		Msg("starting task scheduler service")

	// Schedule job for 00:00 UTC
	_, err := s.scheduler.Cron("0 0 * * *").Do(func() {
		now := time.Now().UTC()
		log.Info().
			Time("run_time", now).
			Msg("triggering scheduled task rescheduling check (00:00 UTC)")

		// Create a new context from the background context for the cron job
		// This ensures the job has its own context that can be traced
		schedulerCtx, span := s.tracer.Start(context.Background(), "task_scheduler.triggered_check",
			trace.WithAttributes(
				attribute.String("run_time", now.Format(time.RFC3339)),
				attribute.String("schedule", "00:00 UTC"),
			))

		if err := s.RunScheduledRescheduling(schedulerCtx); err != nil {
			log.Error().
				Stack().
				Err(err).
				Time("run_time", now).
				Msg("error during scheduled task rescheduling check")
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
		} else {
			span.SetStatus(codes.Ok, "scheduled check completed")
		}
		span.End()
	})
	if err != nil {
		log.Fatal().
			Err(err).
			Msg("failed to schedule task rescheduling job for 00:00 UTC")
		return
	}

	// Schedule job for 12:00 UTC
	_, err = s.scheduler.Cron("0 12 * * *").Do(func() {
		now := time.Now().UTC()
		log.Info().
			Time("run_time", now).
			Msg("triggering scheduled task rescheduling check (12:00 UTC)")

		// Create a new context from the background context for the cron job
		// This ensures the job has its own context that can be traced
		schedulerCtx, span := s.tracer.Start(context.Background(), "task_scheduler.triggered_check",
			trace.WithAttributes(
				attribute.String("run_time", now.Format(time.RFC3339)),
				attribute.String("schedule", "12:00 UTC"),
			))

		if err := s.RunScheduledRescheduling(schedulerCtx); err != nil {
			log.Error().
				Stack().
				Err(err).
				Time("run_time", now).
				Msg("error during scheduled task rescheduling check")
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
		} else {
			span.SetStatus(codes.Ok, "scheduled check completed")
		}
		span.End()
	})
	if err != nil {
		log.Fatal().
			Err(err).
			Msg("failed to schedule task rescheduling job for 12:00 UTC")
		return
	}

	// Start the scheduler
	s.scheduler.StartAsync()
	log.Info().
		Msg("task scheduler started (runs at 00:00 and 12:00 UTC)")

	// Wait for context cancellation
	<-ctx.Done()
	log.Info().
		Msg("task scheduler service stopping")

	// Stop the scheduler
	s.scheduler.Stop()
	log.Info().
		Msg("task scheduler service stopped")
}
