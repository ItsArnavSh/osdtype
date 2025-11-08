package usersession

import (
	"osdtyp/app/entity"
	"osdtyp/app/utils"

	"github.com/gin-gonic/gin"
)

type ActiveSessions struct {
	Users map[uint64]*UserSession
}

func NewActiveSessions() ActiveSessions {
	return ActiveSessions{
		Users: make(map[uint64]*UserSession),
	}
}
func (a *ActiveSessions) GetStatus(id uint64) entity.UserStatus {
	user := a.Users[id]
	if user == nil {
		return entity.OFFLINE
	}
	return user.Status
}
func (a *ActiveSessions) GetSession(id uint64) *UserSession {
	return a.Users[id]
}

func (a *ActiveSessions) NewUserSession(g *gin.Context, id uint64) error {
	ws, err := utils.UpgradeToWebSocket(g)
	if err != nil {
		return err
	}
	a.Users[id] = NewUserSession(ws, a.RemoveSession, id)
	return nil
}
func (a *ActiveSessions) UpdateSession(id uint64, status entity.UserStatus) {
	a.Users[id].Status = status
}

func (a *ActiveSessions) RemoveSession(userID uint64) {
	delete(a.Users, userID)
}
