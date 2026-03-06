package usersession

import (
	"fmt"
	"osdtyp/app/entity"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

type UserSession struct {
	Status           entity.UserStatus
	WS               *websocket.Conn //Not exposing this, communicate through channels
	Incoming         chan []byte
	Outgoing         chan any
	OnDisconnect     func(uint32)
	UserID           uint32
	ChannelShareLock sync.Mutex //Only one process can have the channel at a time
	Logger           *zap.SugaredLogger
	PingLock         sync.Mutex //Only Ping message is allowed to be sent in between other messages
	//Otherwise there is a single sender
	offlineOnce sync.Once
}

func NewUserSession(ws *websocket.Conn, disc func(uint32), id uint32, logger *zap.SugaredLogger) *UserSession {

	fmt.Println("Making new session for ", id)
	user := UserSession{
		Status:           entity.AVAILABLE,
		WS:               ws,
		Incoming:         make(chan []byte),
		Outgoing:         make(chan any),
		OnDisconnect:     disc,
		UserID:           id,
		ChannelShareLock: sync.Mutex{},
	}
	go user.sendData()
	go user.receiveData()
	go user.Ping()
	return &user
}

// Will keep pinging the user
func (u *UserSession) Ping() {
	ticker := time.NewTicker(5 * time.Second)

	for range ticker.C {
		u.PingLock.Lock()
		if err := u.WS.WriteMessage(websocket.PingMessage, nil); err != nil {

			u.UserOffline()
		}
		u.PingLock.Unlock()
	}
}

//Using this pattern to avoid concurrent write on

func (u *UserSession) sendData() { //A running goroutine
	for data := range u.Outgoing {
		if data == nil {
			//Nil data will be sent only when the data wants to unsub from channel
			//Will probably find a better way later, for now, I dont wanna just have UserSession references around the code
			u.UnSubscribe()
			continue
		}
		u.PingLock.Lock()
		err := u.WS.WriteJSON(data)
		if err != nil {
			u.UserOffline()
		}
		u.PingLock.Unlock()
	}
}

// We want to ensure only one goroutine has access to this channel at one time, to avoid bugs
func (u *UserSession) receiveData() { //Keeps filling the channel

	for {
		// Read message from client
		_, message, err := u.WS.ReadMessage()
		if err != nil {
			fmt.Println(err.Error())
			u.UserOffline() //Disconnect in case of error from websocket
			return
		}
		u.Logger.Debugln("Message Len received: ", len(message))
		if len(message) > 0 {
			u.Incoming <- message
		}
	}

}
func (u *UserSession) UserOffline() {
	//To only run it once
	u.offlineOnce.Do(func() {
		fmt.Println("Ending Session")

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
