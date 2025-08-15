package room

import (
	"encoding/json"
	"fmt"
	"osdtype/application/auth"
	"osdtype/application/entity"
	"osdtype/database"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func (g *RoomHandler) create_room(ctx *gin.Context, inst []byte) error {
	user, err := auth.GetUser(ctx)
	if err != nil {
		return err
	}
	var roomdata entity.Room
	err = json.Unmarshal(inst, &roomdata)
	if err != nil {
		return err
	}
	id := uuid.NewString()
	err = g.essentials.Db.CreateRoom(ctx.Request.Context(), database.CreateRoomParams{
		ID:       id,
		AdminID:  user,
		RoomName: roomdata.RoomName,
	})
	if err != nil {
		return err
	}
	return nil
}

func (g *RoomHandler) add_player(ctx *gin.Context, inst []byte) error {
	var playerdata entity.AddPlayer
	err := json.Unmarshal(inst, &playerdata)
	if err != nil {
		return err
	}
	user, err := auth.GetUser(ctx)
	if err != nil {
		return err
	}
	perms, err := g.essentials.Db.CheckUserPerm(ctx.Request.Context(), database.CheckUserPermParams{UserID: user, RoomID: playerdata.RoomID})
	if err != nil {
		return err
	}
	if perms != "admin" && perms != "mod" {
		return fmt.Errorf("Operation Not Permitted")
	}
	//User can perform this action

	err = g.essentials.Db.AddPlayer(ctx.Request.Context(), database.AddPlayerParams{
		UserID:    user,
		RoomID:    playerdata.RoomID,
		PermLevel: "user",
	})
	if err != nil {
		return err
	}
	return nil
}
func (g *RoomHandler) change_player_perms(ctx *gin.Context, inst []byte) error {
	var playerdata entity.ChangePermsStruct
	err := json.Unmarshal(inst, &playerdata)
	if err != nil {
		return err
	}
	user, err := auth.GetUser(ctx)
	if err != nil {
		return err
	}
	perms, err := g.essentials.Db.CheckUserPerm(ctx.Request.Context(), database.CheckUserPermParams{UserID: user, RoomID: playerdata.RoomID})
	if err != nil {
		return err
	}
	modif_perms, err := g.essentials.Db.CheckUserPerm(ctx.Request.Context(), database.CheckUserPermParams{UserID: playerdata.PlayerID, RoomID: playerdata.RoomID})
	if err != nil {
		return err
	}
	if (perms != "admin" && perms != "mod") || (modif_perms == "admin") {
		//Cannot modify if you are user, and cannot affect the admin
		//Only mods and users can be affected by this change
		return fmt.Errorf("Operation Not Permitted")
	}
	err = g.essentials.Db.ChangePlayerPerms(ctx.Request.Context(), database.ChangePlayerPermsParams{
		UserID:    playerdata.PlayerID,
		RoomID:    playerdata.RoomID,
		PermLevel: playerdata.NewPerm,
	})
	if err != nil {
		return err
	}
	return nil
}
func (g *RoomHandler) remove_player(ctx *gin.Context, inst []byte) error {
	//Only admin,mods and the player himself can remove the player
	//And the removed player cannot be the admin
	var playerdata entity.RemovePlayer
	err := json.Unmarshal(inst, &playerdata)
	if err != nil {
		return err
	}
	user, err := auth.GetUser(ctx)
	if err != nil {
		return err
	}
	//Again getting za person and za executioner to compare this stuff
	perms, err := g.essentials.Db.CheckUserPerm(ctx.Request.Context(), database.CheckUserPermParams{UserID: user, RoomID: playerdata.RoomID})
	if err != nil {
		return err
	}
	modif_perms, err := g.essentials.Db.CheckUserPerm(ctx.Request.Context(), database.CheckUserPermParams{UserID: playerdata.PlayerID, RoomID: playerdata.RoomID})
	if err != nil {
		return err
	}
	if (perms != "admin" && perms != "mod") || (modif_perms == "admin") {
		//Cannot modify if you are user, and cannot affect the admin
		//Only mods and users can be affected by this change
		return fmt.Errorf("Operation Not Permitted")
	}
	return g.essentials.Db.RemovePlayer(ctx.Request.Context(), database.RemovePlayerParams{
		UserID: playerdata.PlayerID,
		RoomID: playerdata.RoomID,
	})
}

func (g *GameHandler) ready(_ []byte) error {
	//Needs to be admin or mod
	g.ReadyCompetition()
	return nil
}
func (g *GameHandler) start(ctx *gin.Context, _ []byte) error {
	//Admin or Mod
	g.StartCompetition(ctx.Request.Context())
	return nil
}
func (g *GameHandler) join(_ *gin.Context, _ []byte) error {
	//user
	g.ReadyCompetition()
	return nil
}
func (g *GameHandler) stream(ctx *gin.Context, _ []byte) error {
	//user
	g.SubStream(ctx)
	return nil
}
