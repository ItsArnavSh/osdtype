package usersession

import (
	"fmt"
	"osdtyp/app/entity"
	"osdtyp/app/utils"

	"github.com/gin-gonic/gin"
)

type ActiveSessions struct {
	Users map[uint32]*UserSession
}

func NewActiveSessions() ActiveSessions {
	return ActiveSessions{
		Users: make(map[uint32]*UserSession),
	}
}
func (a *ActiveSessions) GetStatus(id uint32) entity.UserStatus {
	user := a.Users[id]
	if user == nil {
		return entity.OFFLINE
	}
	return user.Status
}
func (a *ActiveSessions) GetSession(id uint32) *UserSession {
	return a.Users[id]
}

func (a *ActiveSessions) NewUserSession(g *gin.Context, id uint32) error {
	ws, err := utils.UpgradeToWebSocket(g)
	if err != nil {
		return err
	}
	a.Users[id] = NewUserSession(ws, a.RemoveSession, id)
	fmt.Println("New Session is here")
	fmt.Print("Total ", len(a.Users))
	for k, v := range a.Users {
		fmt.Println("Member ", k, v)
	}
	return nil
}
func (a *ActiveSessions) UpdateSession(id uint32, status entity.UserStatus) {
	a.Users[id].Status = status
}

func (a *ActiveSessions) RemoveSession(userID uint32) {
	delete(a.Users, userID)
}
