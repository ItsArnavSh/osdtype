package services

import (
	"context"
	"osdtyp/app/api/auth"
	"osdtyp/app/entity"
	"osdtyp/app/utils"

	"github.com/gin-gonic/gin"
)

func (s *ServiceLayer) LoginUser(g *gin.Context, user entity.User) (uint64, error) {
	user_exists, err := s.db.UserExists(g.Request.Context(), user.Username)

	if user_exists {
		user, _ = s.db.GetUserFromName(g.Request.Context(), user.Username)
		return user.ID, nil
	}
	//Register User
	user.ID = s.int_gen.GenerateID()
	user.CurrentRank = 0
	user.AvatarURL, err = utils.GetGitHubAvatar(g.Request.Context(), user.Username)
	if err != nil {
		return 0, err
	}
	err = s.db.AddUser(g.Request.Context(), user)
	if err != nil {
		return 0, err
	}
	auth.SetUserID(g, user.ID)
	return user.ID, nil
}
func (s *ServiceLayer) GetUser(ctx context.Context, userid uint64) (entity.User, error) {
	s.logger.Debug(userid)
	return s.db.GetUser(userid)
}

func (s *ServiceLayer) GetUserFromName(ctx context.Context, username string) (entity.User, error) {
	return s.db.GetUserFromName(ctx, username)
}
