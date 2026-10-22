package scheduler

import (
	"context"
	"osdtyp/app/entity"
	"osdtyp/app/internal/postgresql"
	"time"

	"go.uber.org/zap"
)

// Controller owns contest-specific state transitions.
type Controller interface {
	HandleContestTask(ctx context.Context, task entity.Task) error
}

type Scheduler struct {
	logger     *zap.SugaredLogger
	db         *postgresql.Database
	controller Controller
	interval   time.Duration
}

func New(logger *zap.SugaredLogger, db *postgresql.Database) *Scheduler {
	return &Scheduler{logger: logger, db: db, interval: 500 * time.Millisecond}
}

func (s *Scheduler) SetController(c Controller) { s.controller = c }

func (s *Scheduler) Start(ctx context.Context) {
	ticker := time.NewTicker(s.interval)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				task, err := s.db.PopDueTask(ctx, time.Now().UTC())
				if err != nil {
					s.logger.Errorw("scheduler: pop task", "error", err)
					continue
				}
				if task.ID == 0 {
					continue
				}
				if s.controller == nil {
					continue
				}
				if err := s.controller.HandleContestTask(ctx, task); err != nil {
					s.logger.Errorw("scheduler: handle task", "task", task.ID, "error", err)
				}
			}
		}
	}()
}

func (s *Scheduler) AddTask(ctx context.Context, task *entity.Task) error {
	return s.db.CreateTask(ctx, task)
}
