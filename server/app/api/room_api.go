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
