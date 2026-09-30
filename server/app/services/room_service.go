package services

import (
	"context"
	"errors"
	"math"

	"osdtyp/app/entity"

	"github.com/google/uuid"
)

func (s *ServiceLayer) CreateRoom(ctx context.Context, room entity.Room, requester_id uint32) error {
	room.ID = s.int_gen.GenerateID()
	err := s.db.CreateRoom(ctx, &room)
	if err != nil {
		return err
	}
	var room_user entity.Room_User
	room_user.RoomID = room.ID
	room_user.UserID = requester_id
	room_user.Perm = entity.MOD // The creator of room will be mod by default
	err = s.db.AddMember(ctx, room_user)
	if err != nil {
		return err
	}
	return nil
}
func (s *ServiceLayer) AddMember(ctx context.Context, room_user entity.Room_User) error {
	return s.db.AddMember(ctx, room_user)
}

func (s *ServiceLayer) PromoteToMod(ctx context.Context, room_user entity.Room_User, requester_id uint32) error {
	requester, err := s.db.SeePerms(ctx, entity.Room_User{RoomID: room_user.RoomID, UserID: requester_id})
	if err != nil {
		return err
	}
	if requester.Perm == entity.MOD {
		return s.db.UpdateMembership(ctx,
			entity.Room_User{
				RoomID: room_user.RoomID,
				UserID: room_user.UserID,
				Perm:   entity.MOD,
			})
	}
	s.logger.Error("Not Enough Perms")
	return errors.New("not enough perms")
}

func (s *ServiceLayer) DemoteToMember(ctx context.Context, room_user entity.Room_User, requester_id uint32) error {
	requester, err := s.db.SeePerms(ctx, entity.Room_User{RoomID: room_user.RoomID, UserID: requester_id})
	if err != nil {
		return err
	}
	if requester.Perm == entity.MOD {
		return s.db.UpdateMembership(ctx,
			entity.Room_User{
				RoomID: room_user.RoomID,
				UserID: room_user.UserID,
				Perm:   entity.MEMBER,
			})
	}
	s.logger.Error("Not Enough Perms")
	return errors.New("not enough perms")
}
func (s *ServiceLayer) BlockUser(ctx context.Context, room_user entity.Room_User, requester_id uint32) error {
	requester, err := s.db.SeePerms(ctx, entity.Room_User{RoomID: room_user.RoomID, UserID: requester_id})
	if err != nil {
		return err
	}
	if requester.Perm == entity.MOD {
		return s.db.UpdateMembership(ctx,
			entity.Room_User{
				RoomID: room_user.RoomID,
				UserID: room_user.UserID,
				Perm:   entity.BLOCKED,
			})
	}
	s.logger.Error("Not Enough Perms")
	return errors.New("not enough perms")
}
func (s *ServiceLayer) RemoveUser(ctx context.Context, room_user entity.Room_User, requester_id uint32) error {
	requester, err := s.db.SeePerms(ctx, entity.Room_User{RoomID: room_user.RoomID, UserID: requester_id})
	if err != nil {
		return err
	}

	if requester.Perm == entity.MOD || requester.RoomID == room_user.RoomID { // Either a mod or leaving themselves
		// Getting the perms to maintain BLOCK VS LEFT state
		room_user, err = s.db.SeePerms(ctx, room_user)
		if err != nil {
			return err
		}
		if room_user.Perm == entity.BLOCKED {
			// Dont update his status, he will remain blocked
			return nil
		}
		return s.db.UpdateMembership(ctx,
			entity.Room_User{
				RoomID: room_user.RoomID,
				UserID: room_user.UserID,
				Perm:   entity.LEFT,
			})
	}
	s.logger.Error("Not Enough Perms")
	return errors.New("not enough perms")
}
func (s *ServiceLayer) UnBlockUser(ctx context.Context, room_user entity.Room_User, requester_id uint32) error {
	requester, err := s.db.SeePerms(ctx, entity.Room_User{RoomID: room_user.RoomID, UserID: requester_id})
	if err != nil {
		return err
	}
	if requester.Perm == entity.MOD {
		return s.db.UpdateMembership(ctx,
			entity.Room_User{
				RoomID: room_user.RoomID,
				UserID: room_user.UserID,
				Perm:   entity.MEMBER,
			})
	}
	s.logger.Error("Not Enough Perms")
	return errors.New("not enough perms")
}

// ListRooms returns one page of the rooms a user belongs to. The page index is
// a full uint so pagination is not capped at 255 pages.
func (s *ServiceLayer) ListRooms(ctx context.Context, userID uint32, index uint32) ([]entity.Room, error) {
	const pageSize = 10
	// Keep the offset arithmetic in 64 bits so a large page index cannot wrap
	// into a negative offset.
	if int64(index)*pageSize > math.MaxInt32 {
		return []entity.Room{}, nil
	}
	return s.db.PageList(ctx, userID, int(index), pageSize)
}

// NewContest stores a contest and schedules it.
//
// The contest used to be written with JobID left at zero, and the task was
// queued with JobID zero as well, so every contest shared one job id. The
// scheduler looks the contest up by job id, so the first contest created was
// the one every later job acted on, and UpdateContest's "where job_id = ?"
// matched all of them. A fresh id is now minted and put on both rows.
func (s *ServiceLayer) NewContest(ctx context.Context, contest entity.Contest) error {
	contest.ID = uuid.NewString()
	contest.Status = entity.UPCOMING
	contest.JobID = s.int_gen.GenerateID()

	if err := s.db.NewContest(ctx, contest); err != nil {
		s.logger.Errorw("could not store the contest", "contest", contest.ID, "error", err)
		return err
	}
	s.logger.Infof("Contest in db")

	s.core.Scheduler.NewTask(entity.Task{
		Category: entity.CONTEST,
		JobID:    contest.JobID,
		Time:     contest.Time,
	})

	s.logger.Infow("Added contest to the scheduler", "contest", contest.ID, "job_id", contest.JobID)
	return nil
}
func (s *ServiceLayer) FetchContests(ctx context.Context, room_id uint32, index int) ([]entity.Contest, error) {
	return s.db.ListContests(room_id, index, 50)
}
func (s *ServiceLayer) FetchContestData(ctx context.Context, job_id uint32) (entity.Contest, error) {
	return s.db.GetContestData(job_id)
}
