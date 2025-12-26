package usersession

import (
	"fmt"
	"osdtyp/app/entity"
	"sync"

	"github.com/gorilla/websocket"
)

type UserSession struct {
	Status           entity.UserStatus
	WS               *websocket.Conn //Not exposing this, communicate through channels
	Incoming         chan []byte
	Outgoing         chan any
	OnDisconnect     func(uint32)
	UserID           uint32
	ChannelShareLock sync.Mutex //Only one process can have the channel at a time

}

func NewUserSession(ws *websocket.Conn, disc func(uint32), id uint32) *UserSession {

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
	return &user
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
		err := u.WS.WriteJSON(data)
		if err != nil {
			u.UserOffline()
		}
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
		u.Incoming <- message
	}

}
func (u *UserSession) UserOffline() {
	fmt.Println("Ending Session")
	close(u.Incoming)
	close(u.Outgoing)
	u.OnDisconnect(u.UserID)
}
func (u *UserSession) Subscribe() (<-chan []byte, chan<- any) {
	u.ChannelShareLock.Lock()
	return u.Incoming, u.Outgoing
}

// It MUST be called and channel to stopped being read from from that section of code
func (u *UserSession) UnSubscribe() {
	u.ChannelShareLock.Unlock()
}
