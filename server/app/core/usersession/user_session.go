package usersession

import (
	"sync"
	"sync/atomic"
	"time"

	"osdtyp/app/entity"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

type UserSession struct {
	// status is accessed by the HTTP goroutine handling /user/imonline and by
	// whichever handler asks about a user's availability, so it is an atomic
	// rather than a plain field.
	status atomic.Int32
	WS     *websocket.Conn // Not exposing this, communicate through channels
	// Incoming and Outgoing are closed by UserOffline. sendData ranges over
	// Outgoing and receiveData returns on the read error, so nothing writes to
	// them after the close.
	Incoming         chan []byte
	Outgoing         chan any
	OnDisconnect     func(uint32)
	UserID           uint32
	ChannelShareLock sync.Mutex // Only one process can have the channel at a time
	Logger           *zap.SugaredLogger
	PingLock         sync.Mutex // Only Ping message is allowed to be sent in between other messages
	// Otherwise there is a single sender
	offlineOnce sync.Once
	// closed is signaled once the session is torn down, so the ping loop can
	// exit instead of writing to a dead socket forever.
	closed chan struct{}
}

func NewUserSession(ws *websocket.Conn, disc func(uint32), id uint32, logger *zap.SugaredLogger) *UserSession {
	logger.Infof("Making new session for %d", id)
	user := UserSession{
		WS:               ws,
		Incoming:         make(chan []byte),
		Outgoing:         make(chan any),
		OnDisconnect:     disc,
		UserID:           id,
		ChannelShareLock: sync.Mutex{},
		Logger:           logger,
		closed:           make(chan struct{}),
	}
	user.status.Store(int32(entity.AVAILABLE))
	go user.sendData()
	go user.receiveData()
	go user.Ping()
	return &user
}

// Status reports the user's current availability.
func (u *UserSession) Status() entity.UserStatus {
	return entity.UserStatus(u.status.Load())
}

// SetStatus records the user's current availability.
func (u *UserSession) SetStatus(s entity.UserStatus) {
	u.status.Store(int32(s))
}

// IsClosed reports whether the session has already been torn down.
func (u *UserSession) IsClosed() bool {
	select {
	case <-u.closed:
		return true
	default:
		return false
	}
}

// Will keep pinging the user
func (u *UserSession) Ping() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-u.closed:
			// The ticker used to be left running for the life of the process.
			// Once a user disconnected, this goroutine kept pinging a dead
			// socket every five seconds, and each socket left a goroutine
			// behind.
			return
		case <-ticker.C:
			u.PingLock.Lock()
			err := u.WS.WriteMessage(websocket.PingMessage, nil)
			u.PingLock.Unlock()
			if err != nil {
				u.UserOffline()
				return
			}
		}
	}
}

// Using this pattern to avoid concurrent write on

func (u *UserSession) sendData() { // A running goroutine
	for data := range u.Outgoing {
		if data == nil {
			// Nil data is sent only when a subscriber wants to unsubscribe.
			// A dedicated message would be tidier, but that would mean handing
			// UserSession references to every caller.
			u.UnSubscribe()
			continue
		}
		u.PingLock.Lock()
		err := u.WS.WriteJSON(data)
		u.PingLock.Unlock()
		if err != nil {
			u.UserOffline()
		}
	}
}

// We want to ensure only one goroutine has access to this channel at one time, to avoid bugs
func (u *UserSession) receiveData() { // Keeps filling the channel
	for {
		// Read message from client
		_, message, err := u.WS.ReadMessage()
		if err != nil {
			u.Logger.Debugf("Session read ended: %v", err)
			u.UserOffline() // Disconnect in case of error from websocket
			return
		}
		u.Logger.Debugln("Message Len received: ", len(message))
		if len(message) > 0 {
			u.Incoming <- message
		}
	}
}
func (u *UserSession) UserOffline() {
	// To only run it once
	u.offlineOnce.Do(func() {
		u.Logger.Info("Ending session")

		close(u.closed)
		close(u.Incoming)
		close(u.Outgoing)

		if u.OnDisconnect != nil {
			u.OnDisconnect(u.UserID)
		}
	})
}

// Functions like GameHandler can subscribe to a UserSession, so at that time only they can send/recv messages
// Done for 1) Not sending concurrent messages through WS which is not allowed aaand 2) To reduce bugs
func (u *UserSession) Subscribe() (<-chan []byte, chan<- any) {
	u.ChannelShareLock.Lock()
	return u.Incoming, u.Outgoing
}

// It MUST be called and channel to stopped being read from from that section of code
func (u *UserSession) UnSubscribe() {
	u.ChannelShareLock.Unlock()
}
