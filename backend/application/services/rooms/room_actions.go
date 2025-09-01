package room

import (
	"encoding/json"
	"io"
	"net/http"
	"osdtype/application/auth"
	"osdtype/application/entity"
	"osdtype/database"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func (g *RoomHandler) Create_room(ctx *gin.Context) {
	inst, err := io.ReadAll(ctx.Request.Body)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Failed to read request body"})
		return
	}
	defer ctx.Request.Body.Close()

	user, err := auth.GetUser(ctx)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication failed"})
		return
	}

	var roomdata entity.Room
	err = json.Unmarshal(inst, &roomdata)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON format"})
		return
	}

	id := uuid.NewString()
	err = g.essentials.Db.CreateRoom(ctx.Request.Context(), database.CreateRoomParams{
		ID:       id,
		AdminID:  user,
		RoomName: roomdata.RoomName,
	})
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create room"})
		return
	}

	ctx.JSON(http.StatusCreated, gin.H{"message": "Room created successfully", "room_id": id})
}

func (g *RoomHandler) Add_player(ctx *gin.Context) {
	inst, err := io.ReadAll(ctx.Request.Body)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Failed to read request body"})
		return
	}
	defer ctx.Request.Body.Close()

	var playerdata entity.AddPlayer
	err = json.Unmarshal(inst, &playerdata)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON format"})
		return
	}

	user, err := auth.GetUser(ctx)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication failed"})
		return
	}

	perms, err := g.essentials.Db.CheckUserPerm(ctx.Request.Context(), database.CheckUserPermParams{UserID: user, RoomID: playerdata.RoomID})
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check permissions"})
		return
	}

	if perms != "admin" && perms != "mod" {
		ctx.JSON(http.StatusForbidden, gin.H{"error": "Operation Not Permitted"})
		return
	}

	// User can perform this action
	err = g.essentials.Db.AddPlayer(ctx.Request.Context(), database.AddPlayerParams{
		UserID:    user,
		RoomID:    playerdata.RoomID,
		PermLevel: "user",
	})
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add player"})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "Player added successfully"})
}

func (g *RoomHandler) Change_player_perms(ctx *gin.Context) {
	inst, err := io.ReadAll(ctx.Request.Body)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Failed to read request body"})
		return
	}
	defer ctx.Request.Body.Close()

	var playerdata entity.ChangePermsStruct
	err = json.Unmarshal(inst, &playerdata)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON format"})
		return
	}

	user, err := auth.GetUser(ctx)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication failed"})
		return
	}

	perms, err := g.essentials.Db.CheckUserPerm(ctx.Request.Context(), database.CheckUserPermParams{UserID: user, RoomID: playerdata.RoomID})
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check user permissions"})
		return
	}

	modif_perms, err := g.essentials.Db.CheckUserPerm(ctx.Request.Context(), database.CheckUserPermParams{UserID: playerdata.PlayerID, RoomID: playerdata.RoomID})
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check target user permissions"})
		return
	}

	if (perms != "admin" && perms != "mod") || (modif_perms == "admin") {
		// Cannot modify if you are user, and cannot affect the admin
		// Only mods and users can be affected by this change
		ctx.JSON(http.StatusForbidden, gin.H{"error": "Operation Not Permitted"})
		return
	}

	err = g.essentials.Db.ChangePlayerPerms(ctx.Request.Context(), database.ChangePlayerPermsParams{
		UserID:    playerdata.PlayerID,
		RoomID:    playerdata.RoomID,
		PermLevel: playerdata.NewPerm,
	})
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to change player permissions"})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "Player permissions changed successfully"})
}

func (g *RoomHandler) Remove_player(ctx *gin.Context) {
	// Only admin, mods and the player himself can remove the player
	// And the removed player cannot be the admin
	inst, err := io.ReadAll(ctx.Request.Body)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Failed to read request body"})
		return
	}
	defer ctx.Request.Body.Close()

	var playerdata entity.RemovePlayer
	err = json.Unmarshal(inst, &playerdata)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON format"})
		return
	}

	user, err := auth.GetUser(ctx)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication failed"})
		return
	}

	// Again getting za person and za executioner to compare this stuff
	perms, err := g.essentials.Db.CheckUserPerm(ctx.Request.Context(), database.CheckUserPermParams{UserID: user, RoomID: playerdata.RoomID})
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check user permissions"})
		return
	}

	modif_perms, err := g.essentials.Db.CheckUserPerm(ctx.Request.Context(), database.CheckUserPermParams{UserID: playerdata.PlayerID, RoomID: playerdata.RoomID})
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check target user permissions"})
		return
	}

	if (perms != "admin" && perms != "mod") || (modif_perms == "admin") {
		// Cannot modify if you are user, and cannot affect the admin
		// Only mods and users can be affected by this change
		ctx.JSON(http.StatusForbidden, gin.H{"error": "Operation Not Permitted"})
		return
	}

	err = g.essentials.Db.RemovePlayer(ctx.Request.Context(), database.RemovePlayerParams{
		UserID: playerdata.PlayerID,
		RoomID: playerdata.RoomID,
	})
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to remove player"})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "Player removed successfully"})
}
func (g *GameHandler) ready() {
	g.ReadyCompetition()
}
func (g *GameHandler) start(ctx *gin.Context) {
	//Admin or Mod
	g.StartCompetition(ctx.Request.Context())
}
func (g *GameHandler) join(_ *gin.Context) {
	//user
	g.ReadyCompetition()
}
func (g *GameHandler) stream(ctx *gin.Context) {
	//user
	g.SubStream(ctx)
}
