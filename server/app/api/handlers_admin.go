package api

import (
	"fmt"
	"net/http"
	"strconv"

	"osdtyp/app/api/auth"
	"osdtyp/app/entity"
	"osdtyp/app/services"
	"osdtyp/app/utils"

	"github.com/gin-gonic/gin"
)

const maxAdminOffset = 100000

type adminPageInfo struct {
	Total   int64 `json:"total"`
	Limit   int   `json:"limit"`
	Offset  int   `json:"offset"`
	Next    *int  `json:"next"`
	HasMore bool  `json:"has_more"`
}

func adminPaging(c *gin.Context, defaultLimit, maxLimit int) (int, int) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", strconv.Itoa(defaultLimit)))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	if limit <= 0 {
		limit = defaultLimit
	}
	if limit > maxLimit {
		limit = maxLimit
	}
	if offset < 0 {
		offset = 0
	}
	if offset > maxAdminOffset {
		offset = maxAdminOffset
	}
	return limit, offset
}

func newAdminPage(total int64, limit, offset int) adminPageInfo {
	info := adminPageInfo{
		Total:  total,
		Limit:  limit,
		Offset: offset,
	}
	if next := offset + limit; int64(next) < total {
		info.Next = &next
		info.HasMore = true
	}
	return info
}

func adminPageSlice[T any](items []T, limit, offset int) []T {
	if offset >= len(items) {
		return []T{}
	}
	end := offset + limit
	if end > len(items) {
		end = len(items)
	}
	return items[offset:end]
}

func (s *Server) requireAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Cache-Control", "no-store")
		uid, err := auth.GetUserID(c)
		if err != nil {
			s.abortNotFound(c)
			return
		}
		ok, err := s.services.IsAdmin(c.Request.Context(), uid)
		if err != nil || !ok {
			s.abortNotFound(c)
			return
		}
		c.Set("adminUserID", uid)
		c.Next()
	}
}

func (s *Server) abortNotFound(c *gin.Context) {
	c.AbortWithStatusJSON(http.StatusNotFound, apiError{
		Error:     "not found",
		Code:      "not_found",
		RequestID: requestID(c),
	})
}

func (s *Server) adminActor(c *gin.Context) (entity.User, bool) {
	uid, err := auth.GetUserID(c)
	if err != nil {
		s.abortNotFound(c)
		return entity.User{}, false
	}
	user, err := s.services.GetUser(c.Request.Context(), uid)
	if err != nil {
		s.fail(c, err)
		return entity.User{}, false
	}
	return user, true
}

func (s *Server) adminOverview(c *gin.Context) {
	actor, ok := s.adminActor(c)
	if !ok {
		return
	}
	overview, err := s.services.Overview(c.Request.Context(), actor)
	if err != nil {
		s.fail(c, err)
		return
	}
	c.JSON(http.StatusOK, overview)
}

func (s *Server) adminBugs(c *gin.Context) {
	limit, offset := adminPaging(c, 50, 100)
	filter := entity.BugAdminFilter{
		Status:   c.Query("status"),
		Severity: c.Query("severity"),
		Category: c.Query("category"),
		Query:    c.Query("q"),
	}
	bugs, total, err := s.services.ListBugsAdmin(c.Request.Context(), filter, limit, offset)
	if err != nil {
		s.fail(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"bugs": bugs, "page": newAdminPage(total, limit, offset)})
}

type adminBugUpdateRequest struct {
	Status     string `json:"status"`
	AdminNotes string `json:"admin_notes"`
}

func (s *Server) adminUpdateBug(c *gin.Context) {
	actor, ok := s.adminActor(c)
	if !ok {
		return
	}
	var req adminBugUpdateRequest
	if !bindJSON(c, &req) {
		return
	}
	report, err := s.services.UpdateBugStatus(c.Request.Context(), actor, c.Param("id"), req.Status, req.AdminNotes)
	if err != nil {
		s.fail(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"bug": entity.AdminBugView{
			BugReportView: entity.NewBugReportView(report),
			UserID:        report.UserID,
			AdminNotes:    report.AdminNotes,
		},
	})
}

func (s *Server) adminUsers(c *gin.Context) {
	limit, offset := adminPaging(c, 50, 100)
	filter := entity.UserAdminFilter{Query: c.Query("q"), Role: c.Query("role")}
	users, total, err := s.services.ListUsersAdmin(c.Request.Context(), filter, limit, offset)
	if err != nil {
		s.fail(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"users": users, "page": newAdminPage(total, limit, offset)})
}

func (s *Server) adminGrant(c *gin.Context) {
	actor, ok := s.adminActor(c)
	if !ok {
		return
	}
	target, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		s.fail(c, fmt.Errorf("%w: invalid user id", services.ErrBadRequest))
		return
	}
	if err := s.services.SetUserAdmin(c.Request.Context(), actor, uint32(target), true); err != nil {
		s.fail(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (s *Server) adminRevoke(c *gin.Context) {
	actor, ok := s.adminActor(c)
	if !ok {
		return
	}
	target, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		s.fail(c, fmt.Errorf("%w: invalid user id", services.ErrBadRequest))
		return
	}
	if err := s.services.SetUserAdmin(c.Request.Context(), actor, uint32(target), false); err != nil {
		s.fail(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (s *Server) adminActivity(c *gin.Context) {
	hours, _ := strconv.Atoi(c.DefaultQuery("hours", "24"))
	if hours <= 0 || hours > 168 {
		hours = 24
	}
	limit, offset := adminPaging(c, hours, 168)
	rows, err := s.services.Activity(c.Request.Context(), hours)
	if err != nil {
		s.fail(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"hours": adminPageSlice(rows, limit, offset),
		"page":  newAdminPage(int64(len(rows)), limit, offset),
	})
}

func (s *Server) adminHealth(c *gin.Context) {
	limit, offset := adminPaging(c, 100, 500)
	health, err := s.services.Health(c.Request.Context(), limit, offset)
	if err != nil {
		s.fail(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"current":           health.Current,
		"uptime_24h":        health.Uptime24h,
		"checks_24h":        health.Checks24h,
		"failed_checks_24h": health.FailedChecks24,
		"checks":            health.Checks,
		"page":              newAdminPage(health.Checks24h, limit, offset),
	})
}

func (s *Server) adminLogs(c *gin.Context) {
	limit, offset := adminPaging(c, 100, 200)
	filter := utils.LogFilter{Query: c.Query("q"), Level: c.Query("level"), User: c.Query("user")}
	logs, total := s.services.Logs(c.Request.Context(), filter, limit, offset)
	c.JSON(http.StatusOK, gin.H{"logs": logs, "page": newAdminPage(int64(total), limit, offset)})
}

func (s *Server) adminEvents(c *gin.Context) {
	limit, offset := adminPaging(c, 50, 200)
	filter := entity.SystemEventFilter{Level: c.Query("level"), Kind: c.Query("kind"), Query: c.Query("q")}
	events, total, err := s.services.Events(c.Request.Context(), filter, limit, offset)
	if err != nil {
		s.fail(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"events": events, "page": newAdminPage(total, limit, offset)})
}

func (s *Server) adminAudits(c *gin.Context) {
	limit, offset := adminPaging(c, 50, 200)
	filter := entity.AuditFilter{Actor: c.Query("actor"), Action: c.Query("action"), Query: c.Query("q")}
	audits, total, err := s.services.Audits(c.Request.Context(), filter, limit, offset)
	if err != nil {
		s.fail(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"audits": audits, "page": newAdminPage(total, limit, offset)})
}
