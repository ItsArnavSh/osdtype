package usersession

import (
	"osdtyp/app/entity"
	"sync"

	"github.com/gorilla/websocket"
)

type UserSession struct {
	Status           entity.UserStatus
	WS               *websocket.Conn //Not exposing this, communicate through channels
	Incoming         chan []byte
	Outgoing         chan any
	OnDisconnect     func(uint64)
	UserID           uint64
	lock             sync.RWMutex
	ReadChannelFree  bool
	WriteChannelFree bool
}

//Using this pattern to avoid concurrent write on

func (u *UserSession) SendData() { //A running goroutine
	for data := range u.Outgoing {
		_ = u.WS.WriteJSON(data)
		//If error is related to disconnect, close the channel and handle everything
	}
}

// We want to ensure only one goroutine has access to this channel at one time, to avoid bugs
func (u *UserSession) ReceiveData() { //Keeps filling the channel

	for {
		// Read message from client
		_, message, err := u.WS.ReadMessage()
		if err != nil {
			u.UserOffline() //Disconnect in case of error from websocket
		}
		u.Incoming <- message
	}

}
func (u *UserSession) UserOffline() {
	close(u.Incoming)
	close(u.Outgoing)
	u.OnDisconnect(u.UserID)
}
func (u *UserSession) SubscribeRead() chan []byte {
	u.lock.Lock()
	u.ReadChannelFree = false
	u.lock.Unlock()

	return u.Incoming
}

// It MUST be called and channel to stopped being read from from that section of code
func (u *UserSession) UnSubRead() {
	u.lock.Lock()
	u.ReadChannelFree = true
	u.lock.Unlock()
}

func (u *UserSession) SubscribeWrite() chan any {
	u.lock.Lock()
	u.WriteChannelFree = false
	u.lock.Unlock()

	return u.Outgoing
}

// It MUST be called and channel to stopped being written from from that section of code
func (u *UserSession) UnSubWrite() {
	u.lock.Lock()
	u.WriteChannelFree = true
	u.lock.Unlock()
}
