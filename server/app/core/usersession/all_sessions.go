package usersession

import (
	"fmt"
	"osdtyp/app/entity"
	"osdtyp/app/utils"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

type ActiveSessions struct {
	Users map[uint64]*UserSession
}

func (a *ActiveSessions) GetWS(id uint64) (*websocket.Conn, error) {
	user := a.Users[id]
	if user == nil {
		return nil, fmt.Errorf("Could not find session")
	}
	return user.WS, nil
}
func (a *ActiveSessions) GetStatus(id uint64) entity.UserStatus {
	user := a.Users[id]
	if user == nil {
		return entity.OFFLINE
	}
	return user.Status
}
func (a *ActiveSessions) NewUserSession(g *gin.Context, id uint64) error {
	ws, err := utils.UpgradeToWebSocket(g)
	if err != nil {
		return err
	}
	a.Users[id] = &UserSession{WS: ws, Status: entity.AVAILABLE, OnDisconnect: a.RemoveSession}
	return nil
}
func (a *ActiveSessions) UpdateSession(id uint64, status entity.UserStatus) {
	a.Users[id].Status = status
}

func (a *ActiveSessions) RemoveSession(userID uint64) {
	delete(a.Users, userID)
}
