package api

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"osdtyp/app/api/auth"
	"osdtyp/app/services"

	"github.com/gin-gonic/gin"
)

func (s *Server) listContests(c *gin.Context) {
	uid, _ := auth.GetUserID(c)
	view, err := s.services.ListContests(c, uid)
	if err != nil {
		s.fail(c, err)
		return
	}
	c.JSON(http.StatusOK, view)
}

func (s *Server) getContest(c *gin.Context) {
	uid, _ := auth.GetUserID(c)
	view, err := s.services.GetContestByCode(c, uid, c.Param("code"))
	if err != nil {
		s.fail(c, err)
		return
	}
	c.JSON(http.StatusOK, view)
}

func (s *Server) createContest(c *gin.Context) {
	uid, err := auth.GetUserID(c)
	if err != nil {
		s.fail(c, err)
		return
	}
	var req services.CreateContestRequest
	if !bindJSON(c, &req) {
		return
	}
	view, err := s.services.CreateContest(c, uid, req)
	if err != nil {
		s.fail(c, err)
		return
	}
	c.JSON(http.StatusCreated, view)
}

func (s *Server) updateContest(c *gin.Context) {
	uid, err := auth.GetUserID(c)
	if err != nil {
		s.fail(c, err)
		return
	}
	var req services.UpdateContestRequest
	if !bindJSON(c, &req) {
		return
	}
	view, err := s.services.UpdateContest(c, uid, c.Param("code"), req)
	if err != nil {
		s.fail(c, err)
		return
	}
	c.JSON(http.StatusOK, view)
}

func (s *Server) cancelContest(c *gin.Context) {
	uid, err := auth.GetUserID(c)
	if err != nil {
		s.fail(c, err)
		return
	}
	view, err := s.services.CancelContest(c, uid, c.Param("code"))
	if err != nil {
		s.fail(c, err)
		return
	}
	c.JSON(http.StatusOK, view)
}

func (s *Server) enrollContest(c *gin.Context) {
	uid, err := auth.GetUserID(c)
	if err != nil {
		s.fail(c, err)
		return
	}
	var req struct {
		Password string `json:"password"`
	}
	if c.Request.ContentLength > 0 && !bindJSON(c, &req) {
		return
	}
	view, err := s.services.EnrollContest(c, uid, c.Param("code"), req.Password)
	if err != nil {
		s.fail(c, err)
		return
	}
	c.JSON(http.StatusOK, view)
}

func (s *Server) leaveContest(c *gin.Context) {
	uid, err := auth.GetUserID(c)
	if err != nil {
		s.fail(c, err)
		return
	}
	view, err := s.services.LeaveContest(c, uid, c.Param("code"))
	if err != nil {
		s.fail(c, err)
		return
	}
	c.JSON(http.StatusOK, view)
}

func (s *Server) startContestAttempt(c *gin.Context) {
	uid, err := auth.GetUserID(c)
	if err != nil {
		s.fail(c, err)
		return
	}
	position, err := strconv.Atoi(c.Param("position"))
	if err != nil || position < 1 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid stage"})
		return
	}
	var req services.ContestAttemptRequest
	if c.Request.ContentLength > 0 && !bindJSON(c, &req) {
		return
	}
	view, err := s.services.StartContestAttempt(c, uid, c.Param("code"), position, req)
	if err != nil {
		s.fail(c, err)
		return
	}
	c.JSON(http.StatusCreated, view)
}

func (s *Server) contestCalendar(c *gin.Context) {
	view, err := s.services.GetContestByCode(c, 0, c.Param("code"))
	if err != nil {
		s.fail(c, err)
		return
	}
	if view.Status == "cancelled" {
		c.JSON(http.StatusGone, gin.H{"error": "contest was cancelled"})
		return
	}
	start := view.StartsAt.UTC()
	end := start.Add(time.Duration(view.Duration) * time.Second)
	description := view.Description
	if description == "" {
		description = fmt.Sprintf("%ds %s contest on OSDTYP", view.Duration, view.Language)
	}
	ics := strings.Join([]string{
		"BEGIN:VCALENDAR",
		"VERSION:2.0",
		"PRODID:-//OSDTYP//Contests//EN",
		"CALSCALE:GREGORIAN",
		"METHOD:PUBLISH",
		"BEGIN:VEVENT",
		"UID:" + escapeICS(view.Code+"@osdtyp"),
		"DTSTAMP:" + time.Now().UTC().Format("20060102T150405Z"),
		"DTSTART:" + start.Format("20060102T150405Z"),
		"DTEND:" + end.Format("20060102T150405Z"),
		"SUMMARY:" + escapeICS(view.Title),
		"DESCRIPTION:" + escapeICS(description),
		"URL:" + escapeICS(view.ShareLink),
		"STATUS:CONFIRMED",
		"END:VEVENT",
		"END:VCALENDAR",
		"",
	}, "\r\n")
	c.Header("Content-Type", "text/calendar; charset=utf-8")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%q", view.Code+".ics"))
	c.String(http.StatusOK, ics)
}

func escapeICS(value string) string {
	value = strings.ReplaceAll(value, `\`, `\\`)
	value = strings.ReplaceAll(value, ";", `\;`)
	value = strings.ReplaceAll(value, ",", `\,`)
	value = strings.ReplaceAll(value, "\r\n", `\n`)
	value = strings.ReplaceAll(value, "\n", `\n`)
	return value
}
