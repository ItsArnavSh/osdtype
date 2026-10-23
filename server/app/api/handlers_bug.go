package api

import (
	"net/http"

	"osdtyp/app/api/auth"
	"osdtyp/app/entity"
	"osdtyp/app/services"

	"github.com/gin-gonic/gin"
)

func (s *Server) createBugReport(c *gin.Context) {
	var req services.BugReportInput
	if !bindJSON(c, &req) {
		return
	}
	var userID *uint32
	if id, err := auth.GetUserID(c); err == nil {
		userID = &id
	}
	report, duplicate, err := s.services.CreateBugReport(c.Request.Context(), req, userID, c.ClientIP())
	if err != nil {
		s.fail(c, err)
		return
	}
	status := http.StatusCreated
	if duplicate {
		status = http.StatusOK
	}
	c.JSON(status, gin.H{
		"report":    entity.NewBugReportView(*report),
		"duplicate": duplicate,
	})
}

func (s *Server) listMyBugReports(c *gin.Context) {
	userID, err := auth.GetUserID(c)
	if err != nil {
		s.fail(c, err)
		return
	}
	reports, err := s.services.ListMyBugReports(c.Request.Context(), userID, 50)
	if err != nil {
		s.fail(c, err)
		return
	}
	views := make([]entity.BugReportView, 0, len(reports))
	for _, report := range reports {
		views = append(views, entity.NewBugReportView(report))
	}
	c.JSON(http.StatusOK, gin.H{"reports": views})
}
