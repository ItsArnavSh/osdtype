package api

import (
	"errors"
	"net/http"

	"osdtyp/app/services"

	"github.com/gin-gonic/gin"
)

type apiError struct {
	Error string `json:"error"`
	Code  string `json:"code,omitempty"`
}

func errorStatus(err error) (int, string) {
	switch {
	case errors.Is(err, services.ErrBadRequest):
		return http.StatusBadRequest, "bad_request"
	case errors.Is(err, services.ErrUnauthorized):
		return http.StatusUnauthorized, "unauthorized"
	case errors.Is(err, services.ErrForbidden):
		return http.StatusForbidden, "forbidden"
	case errors.Is(err, services.ErrNotFound):
		return http.StatusNotFound, "not_found"
	case errors.Is(err, services.ErrConflict):
		return http.StatusConflict, "conflict"
	case errors.Is(err, services.ErrPasswordRequired):
		return http.StatusForbidden, "password_required"
	case errors.Is(err, services.ErrWrongPassword):
		return http.StatusForbidden, "wrong_password"
	case errors.Is(err, services.ErrFull):
		return http.StatusConflict, "room_full"
	case errors.Is(err, services.ErrBanned):
		return http.StatusForbidden, "banned"
	}
	return http.StatusInternalServerError, "internal_error"
}

func (s *Server) fail(c *gin.Context, err error) {
	status, code := errorStatus(err)
	msg := err.Error()
	if status == http.StatusInternalServerError {
		s.logger.Errorw("request failed", "path", c.FullPath(), "error", err)
		msg = "internal server error"
	}
	c.AbortWithStatusJSON(status, apiError{Error: msg, Code: code})
}

func bindJSON(c *gin.Context, dst any) bool {
	if err := c.ShouldBindJSON(dst); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, apiError{Error: "invalid JSON body", Code: "bad_request"})
		return false
	}
	return true
}
