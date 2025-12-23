package scheduler

import (
	controlledlobby "osdtyp/app/core/controlled-lobby"
	"osdtyp/app/entity"
	"osdtyp/app/internal/postgresql"
	"time"

	"go.uber.org/zap"
)

/*
 *
 * We will first of all maintain a var NextJob which basically stores the data for the next task
 * Everytime something is passed thru the backend, its first checked if it can be that
 * If yes, the var is updated, the data is saved in the db nonetheless
 * After the NextJob actually takes place, we check the db for the most recent job
 * If there are multiple, pick them all up in an array and start them at set time
 *
 */

// For Scheduling the lobby and organizing the competition
type Scheduler struct {
	logger   *zap.SugaredLogger
	mro      entity.Task //Most recent object idk
	db       *postgresql.Database
	lobby    *controlledlobby.ControlledLobby
	wakechan chan entity.Task
}

func (s *Scheduler) StartScheduler() {
	for {
		mro, err := s.db.GetRecentTask()
		if err != nil {
			s.logger.Error("Scheduler has crashed", err)
			return
		}
		if mro.JobID == 0 {
			<-s.wakechan
			continue
			//Sleep till new tasks come
		}
		sleepDuration := time.Until(mro.Time)

		select {
		case <-time.After(sleepDuration):
			s.TaskHandler(mro)
		case task := <-s.wakechan:
			s.db.NewTask(task)
		}
	}
}
func (s *Scheduler) TaskHandler(task entity.Task) {
	switch task.Category {
	case entity.CONTEST:
		lobby_id := s.lobby.CreateNewLobby()

		time.After(time.Until(task.Time) - time.Minute*5)
		s.lobby.StartGameFromLobby()
		//Since it is a general purpose scheduler, other entries can be simply added later on
	}
}
func (s *Scheduler) NewTask(task entity.Task) {
	//Wakey Wakey
	s.wakechan <- task
}
func NewScheduler(logger *zap.SugaredLogger, db *postgresql.Database, ml *controlledlobby.ControlledLobby) (Scheduler, error) {

	return Scheduler{
		logger: logger,
		db:     db,
		mro:    entity.Task{},
		lobby:  ml,
	}, nil
}
