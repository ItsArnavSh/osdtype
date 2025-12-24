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
		mro, err := s.db.PopRecentTask()
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
		contest, err := s.db.GetContestData(task.JobID)
		if err != nil {
			return
		}
		switch contest.Status {
		case entity.UPCOMING:
			lobby_id := s.lobby.CreateNewLobby()
			contest.LobbyID = lobby_id
			contest.Status = entity.LOBBY
			err = s.db.UpdateContest(contest)
			if err != nil {
				return
			}
			task.Time = task.Time.Add(5 * time.Minute) //Put back in scheduler so that it can now run the game
			err := s.db.NewTask(task)
			if err != nil {
				return //Todo: Do better error handling over here
			}
		//Update the task time
		case entity.LOBBY:
			sig := make(chan []entity.WPMRes)
			s.lobby.StartGameFromLobby(contest.LobbyID, contest.Duration.Duration(), sig)
			contest.Status = entity.STARTED
			err = s.db.UpdateContest(contest)
			select {
			case leaderboard := <-sig:
				contest.Status = entity.ENDED
				contest.Leaderboard = leaderboard
				err := s.db.UpdateContest(contest)
				if err != nil {
					return
				}
			case <-time.After(10 * time.Minute):
				return //Timeout
			}

		}
	}
	//Since it is a general purpose scheduler, other entries can be simply added later on
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
