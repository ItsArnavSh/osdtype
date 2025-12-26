package api

import (
	"encoding/json"
	"io"
	"net/http"
	"osdtyp/app/api/auth"
	"osdtyp/app/entity"
	"strconv"

	"github.com/gin-gonic/gin"
)

func (s *Server) CreateRoom(c *gin.Context) {
	s.logger.Infof("Creating new Room")
	jsonData, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Error Reading JSON"})
	}
	var room_data entity.Room
	err = json.Unmarshal(jsonData, &room_data)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Error Parsing JSON"})
	}
	user_id, err := auth.GetUserID(c)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "User not logged in"})
	}
	err = s.services.CreateRoom(c.Request.Context(), room_data, user_id)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": err})
	}
}
func (s *Server) AddMember(c *gin.Context) {
	s.logger.Infof("Adding member to Room")

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
		c.JSON(http.StatusForbidden, gin.H{"error": "User not logged in"})
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
		c.JSON(http.StatusForbidden, gin.H{"error": "User not logged in"})
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
		c.JSON(http.StatusForbidden, gin.H{"error": "User not logged in"})
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
		c.JSON(http.StatusForbidden, gin.H{"error": "User not logged in"})
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
		c.JSON(http.StatusForbidden, gin.H{"error": "User not logged in"})
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
	user_id, err := auth.GetUserID(c)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "User not logged in"})
		return
	}
	index_str := c.Query("index")
	index, err := strconv.ParseUint(index_str, 10, 8)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Error in query"})
		return
	}
	rooms, err := s.services.ListRooms(c.Request.Context(), user_id, uint8(index))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error"})
		return
	}
	rooms_json, err := json.Marshal(rooms)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Unable to marshal"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"rooms": rooms_json})
}
func (s *Server) CreateContest(c *gin.Context) {

	var contest entity.Contest
	if err := c.ShouldBindJSON(&contest); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	// Optional: you might want to set the creator or validate permissions here
	// contest.CreatorID = user_id

	err := s.services.NewContest(c.Request.Context(), contest)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create contest"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Contest created and scheduled successfully"})
}

func (s *Server) GetContests(c *gin.Context) {

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

	contests, err := s.services.FetchContests(c.Request.Context(), uint32(roomID), int(index))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch contests"})
		return
	}

	contestsJSON, err := json.Marshal(contests)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Unable to marshal contests"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"contests": contestsJSON})
}

func (s *Server) GetContestData(c *gin.Context) {
	jobIDStr := c.Param("job_id") // assuming you use /contests/:job_id route
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
		// You might want to distinguish not-found vs server error
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch contest data"})
		return
	}

	contestJSON, err := json.Marshal(contest)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Unable to marshal contest data"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"contest": contestJSON})
}
