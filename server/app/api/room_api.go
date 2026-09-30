package api

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"

	"osdtyp/app/api/auth"
	"osdtyp/app/entity"

	"github.com/gin-gonic/gin"
)

func (s *Server) CreateRoom(c *gin.Context) {
	s.logger.Infof("Creating new Room")

	// AuthMiddleware is advisory, so the handler has to verify the caller.
	userID, err := auth.GetUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not logged in"})
		return
	}

	jsonData, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Error Reading JSON"})
		return
	}

	var roomData entity.Room
	if err := json.Unmarshal(jsonData, &roomData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Error Parsing JSON"})
		return
	}

	if err := s.services.CreateRoom(c.Request.Context(), roomData, userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "success"})
}
func (s *Server) AddMember(c *gin.Context) {
	s.logger.Infof("Adding member to Room")

	// This handler used to skip the caller check entirely, so anybody could
	// add themselves to any room.
	if _, err := auth.GetUserID(c); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not logged in"})
		return
	}

	jsonData, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Error Reading JSON"})
		return
	}

	var room_user entity.Room_User
	err = json.Unmarshal(jsonData, &room_user)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Error Parsing JSON"})
		return
	}

	err = s.services.AddMember(c.Request.Context(), room_user)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "success"})
}

func (s *Server) PromoteToMod(c *gin.Context) {
	s.logger.Infof("Promoting user to Mod")

	jsonData, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Error Reading JSON"})
		return
	}

	var room_user entity.Room_User
	err = json.Unmarshal(jsonData, &room_user)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Error Parsing JSON"})
		return
	}

	user_id, err := auth.GetUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not logged in"})
		return
	}

	err = s.services.PromoteToMod(c.Request.Context(), room_user, user_id)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "success"})
}

func (s *Server) DemoteToMember(c *gin.Context) {
	s.logger.Infof("Demoting user to Member")

	jsonData, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Error Reading JSON"})
		return
	}

	var room_user entity.Room_User
	err = json.Unmarshal(jsonData, &room_user)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Error Parsing JSON"})
		return
	}

	user_id, err := auth.GetUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not logged in"})
		return
	}

	err = s.services.DemoteToMember(c.Request.Context(), room_user, user_id)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "success"})
}

func (s *Server) BlockUser(c *gin.Context) {
	s.logger.Infof("Blocking user from Room")

	jsonData, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Error Reading JSON"})
		return
	}

	var room_user entity.Room_User
	err = json.Unmarshal(jsonData, &room_user)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Error Parsing JSON"})
		return
	}

	user_id, err := auth.GetUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not logged in"})
		return
	}

	err = s.services.BlockUser(c.Request.Context(), room_user, user_id)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "success"})
}

func (s *Server) RemoveUser(c *gin.Context) {
	s.logger.Infof("Blocking user from Room")

	jsonData, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Error Reading JSON"})
		return
	}

	var room_user entity.Room_User
	err = json.Unmarshal(jsonData, &room_user)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Error Parsing JSON"})
		return
	}

	user_id, err := auth.GetUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not logged in"})
		return
	}

	err = s.services.RemoveUser(c.Request.Context(), room_user, user_id)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "success"})
}

func (s *Server) UnBlockUser(c *gin.Context) {
	s.logger.Infof("Blocking user from Room")

	jsonData, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Error Reading JSON"})
		return
	}

	var room_user entity.Room_User
	err = json.Unmarshal(jsonData, &room_user)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Error Parsing JSON"})
		return
	}

	user_id, err := auth.GetUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not logged in"})
		return
	}

	err = s.services.UnBlockUser(c.Request.Context(), room_user, user_id)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "success"})
}

func (s *Server) GetRoomList(c *gin.Context) {
	userID, err := auth.GetUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not logged in"})
		return
	}

	indexStr := c.Query("index")
	if indexStr == "" {
		indexStr = "0"
	}
	// The page index was parsed with a bit size of 8, which capped pagination
	// at page 255 and made every later page unreachable. A page index is small
	// but unbounded in practice, so it is parsed as a full uint.
	index, err := strconv.ParseUint(indexStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid index"})
		return
	}

	rooms, err := s.services.ListRooms(c.Request.Context(), userID, uint32(index))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error"})
		return
	}

	// Pass the slice to gin directly. Marshaling it first and putting the
	// []byte in the response made encoding/json emit a base64 string.
	c.JSON(http.StatusOK, gin.H{"rooms": rooms})
}
func (s *Server) CreateContest(c *gin.Context) {
	// AuthMiddleware is advisory: it populates the context when a token is
	// present but never rejects the request. Every handler behind it has to
	// check for itself. The contest routes used to skip this, so an anonymous
	// caller could schedule contests.
	if _, err := auth.GetUserID(c); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not logged in"})
		return
	}

	var contest entity.Contest
	if err := c.ShouldBindJSON(&contest); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	err := s.services.NewContest(c.Request.Context(), contest)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create contest"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Contest created and scheduled successfully"})
}

func (s *Server) GetContests(c *gin.Context) {
	if _, err := auth.GetUserID(c); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not logged in"})
		return
	}

	roomIDStr := c.Query("room_id")
	if roomIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "room_id query parameter is required"})
		return
	}
	roomID, err := strconv.ParseUint(roomIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid room_id"})
		return
	}

	indexStr := c.Query("index")
	index, err := strconv.ParseInt(indexStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid index query parameter"})
		return
	}
	if index < 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid index query parameter"})
		return
	}

	contests, err := s.services.FetchContests(c.Request.Context(), uint32(roomID), int(index))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch contests"})
		return
	}

	// Hand the slice straight to gin's encoder. Marshaling it here and
	// putting the resulting []byte in the response made encoding/json emit a
	// base64 string, which is not what the client expects.
	c.JSON(http.StatusOK, gin.H{"contests": contests})
}

func (s *Server) GetContestData(c *gin.Context) {
	if _, err := auth.GetUserID(c); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not logged in"})
		return
	}

	jobIDStr := c.Param("job_id")
	if jobIDStr == "" {
		jobIDStr = c.Query("job_id") // fallback to query param if needed
	}
	if jobIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "job_id is required"})
		return
	}

	jobID, err := strconv.ParseUint(jobIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid job_id"})
		return
	}

	contest, err := s.services.FetchContestData(c.Request.Context(), uint32(jobID))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Contest not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"contest": contest})
}
