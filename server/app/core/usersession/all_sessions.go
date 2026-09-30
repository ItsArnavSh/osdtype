package usersession

import (
	"sync"

	"osdtyp/app/entity"
	"osdtyp/app/utils"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// ActiveSessions is the registry of every user currently connected over a
// WebSocket.
//
// Sessions are created and torn down on HTTP handler goroutines while the
// matchmaker reads them from its own worker, so the map is guarded. It used to
// be a plain map field, which meant a connect or a disconnect racing a lookup
// could take the whole process down with "concurrent map read and map write".
type ActiveSessions struct {
	mu     sync.RWMutex
	users  map[uint32]*UserSession
	logger *zap.SugaredLogger
}

func NewActiveSessions(logger *zap.SugaredLogger) ActiveSessions {
	return ActiveSessions{
		users:  make(map[uint32]*UserSession),
		logger: logger,
	}
}

// Count reports how many users are currently connected.
func (a *ActiveSessions) Count() int {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return len(a.users)
}

// store registers a session without going through a WebSocket upgrade. Used by
// NewUserSession and by tests, which cannot perform a real handshake.
func (a *ActiveSessions) store(id uint32, session *UserSession) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.users[id] = session
	a.logger.Infof("Session opened for user %d (%d connected)", id, len(a.users))
}

func (a *ActiveSessions) GetStatus(id uint32) entity.UserStatus {
	user := a.GetSession(id)
	if user == nil {
		return entity.OFFLINE
	}
	return user.Status()
}

func (a *ActiveSessions) GetSession(id uint32) *UserSession {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.users[id]
}

func (a *ActiveSessions) NewUserSession(g *gin.Context, id uint32) error {
	ws, err := utils.UpgradeToWebSocket(g)
	if err != nil {
		return err
	}
	session := NewUserSession(ws, a.RemoveSession, id, a.logger)
	a.store(id, session)
	return nil
}

func (a *ActiveSessions) UpdateSession(id uint32, status entity.UserStatus) {
	// A user can disconnect between the caller deciding they are online and
	// this write landing, so a missing session is expected rather than a bug.
	session := a.GetSession(id)
	if session == nil {
		return
	}
	session.SetStatus(status)
}

func (a *ActiveSessions) RemoveSession(userID uint32) {
	a.mu.Lock()
	delete(a.users, userID)
	count := len(a.users)
	a.mu.Unlock()

	a.logger.Infof("Session closed for user %d (%d connected)", userID, count)
}
