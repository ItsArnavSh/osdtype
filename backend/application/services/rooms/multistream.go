package room

import (
	"context"
	"osdtype/application/entity"
	"sync"
)

func (R *RoomHandler) PlayersReady(ctx context.Context) {

}
func (R *RoomHandler) StartTyping(ctx context.Context, wg *sync.WaitGroup, keychan chan entity.KeyDef) {

}

//So they will be sharing a channel pretty much getting the updates of all the users
// And run in parallel

func (R *RoomHandler) AddViewer() {
	//Send the whole currently typed scene of all members, then add him in the viewers list (Also look for locking mechanisms to do so)
}
func (R *RoomHandler) ViewerStream() {
	// Fetch the viewers list everytime from the class def, so that it can be updated parallely.
}
