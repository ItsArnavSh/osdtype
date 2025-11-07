package services

import (
	"context"
	"osdtyp/app/entity"
	"osdtyp/app/utils"
)

func (s *ServiceLayer) LoginUser(ctx context.Context, user entity.User) error {
	user_exists, err := s.db.UserExists(ctx, user.Username)
	if err != nil {
		return err
	}
	if user_exists {
		return nil //Already exists in database
	}
	//Register User
	user.ID = s.int_gen.GenerateID()
	user.CurrentRank = 0
	user.AvatarURL, err = utils.GetGitHubAvatar(ctx, user.Username)
	if err != nil {
		return err
	}
	err = s.db.AddUser(ctx, user)
	if err != nil {
		return err
	}
	return nil
}
func (s *ServiceLayer) GetUser(ctx context.Context, username string) (entity.User, error) {
	return s.db.GetUser(ctx, username)
}
