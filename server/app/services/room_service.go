package services

import (
	"context"
	"fmt"
	"osdtyp/app/entity"
)

func (s *ServiceLayer) CreateRoom(ctx context.Context, room entity.Room, requester_id uint64) error {
	room.ID = s.int_gen.GenerateID()
	err := s.db.CreateRoom(ctx, room)
	if err != nil {
		return err
	}
	var room_user entity.Room_User
	room_user.RoomID = room.ID
	room_user.UserID = requester_id
	room_user.Perm = entity.MOD //The creator of room will be mod by default
	err = s.db.AddMember(ctx, room_user)
	if err != nil {
		return err
	}
	return nil
}
func (s *ServiceLayer) AddMember(ctx context.Context, room_user entity.Room_User) error {
	return s.db.AddMember(ctx, room_user)
}

func (s *ServiceLayer) PromoteToMod(ctx context.Context, room_user entity.Room_User, requester_id uint64) error {
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
	return fmt.Errorf("Not enough perms")
}

func (s *ServiceLayer) DemoteToMember(ctx context.Context, room_user entity.Room_User, requester_id uint64) error {
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
	return fmt.Errorf("Not enough perms")
}
func (s *ServiceLayer) BlockUser(ctx context.Context, room_user entity.Room_User, requester_id uint64) error {
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
	return fmt.Errorf("Not enough perms")
}
func (s *ServiceLayer) RemoveUser(ctx context.Context, room_user entity.Room_User, requester_id uint64) error {
	requester, err := s.db.SeePerms(ctx, entity.Room_User{RoomID: room_user.RoomID, UserID: requester_id})
	if err != nil {
		return err
	}

	if requester.Perm == entity.MOD || requester.RoomID == room_user.RoomID { //Either a mod or leaving themselves
		//Getting the perms to maintain BLOCK VS LEFT state
		room_user, err = s.db.SeePerms(ctx, room_user)
		if err != nil {
			return err
		}
		if room_user.Perm == entity.BLOCKED {
			//Dont update his status, he will remain blocked
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
	return fmt.Errorf("Not enough perms")
}
func (s *ServiceLayer) UnBlockUser(ctx context.Context, room_user entity.Room_User, requester_id uint64) error {
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
	return fmt.Errorf("Not enough perms")
}

func (s *ServiceLayer) ListRooms(ctx context.Context, user_id uint64, index uint8) ([]entity.Room, error) {
	return s.db.PageList(ctx, user_id, index, 10)
}
