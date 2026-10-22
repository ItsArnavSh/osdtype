package api

import (
	"bytes"
	"context"
	"net/http"
	"strconv"
	"time"

	"osdtyp/app/api/auth"
	"osdtyp/app/entity"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// captureWriter records the response while still writing it to the client.
type captureWriter struct {
	gin.ResponseWriter
	body bytes.Buffer
}

func (w *captureWriter) Write(b []byte) (int, error) {
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}

func (w *captureWriter) WriteString(s string) (int, error) {
	w.body.WriteString(s)
	return w.ResponseWriter.WriteString(s)
}

// IdempotencyKey support is opt-in: only requests that carry the header are
// recorded. A replay returns the exact original body and status.
func (s *Server) idempotency() gin.HandlerFunc {
	return func(c *gin.Context) {
		key := c.GetHeader("Idempotency-Key")
		if key == "" || c.Request.Method == http.MethodGet || c.Request.Method == http.MethodOptions {
			c.Next()
			return
		}
		uid, err := auth.GetUserID(c)
		if err != nil {
			c.Next()
			return
		}
		scoped := "idem:" + strconv.FormatUint(uint64(uid), 10) + ":" + key
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		row, err := s.db.GetIdempotency(ctx, scoped)
		cancel()
		if err == nil && len(row.Response) > 0 {
			c.Header("Idempotent-Replay", "true")
			c.Data(row.Status, "application/json; charset=utf-8", row.Response)
			c.Abort()
			return
		}
		if err != nil && err != gorm.ErrRecordNotFound {
			s.logger.Warnw("idempotency lookup failed", "error", err)
		}

		cw := &captureWriter{ResponseWriter: c.Writer}
		c.Writer = cw
		c.Next()
		c.Writer = cw.ResponseWriter

		if cw.Status() < 200 || cw.Status() >= 300 {
			return
		}
		saveCtx, saveCancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer saveCancel()
		body := append([]byte(nil), cw.body.Bytes()...)
		if err := s.db.SaveIdempotency(saveCtx, &entity.IdempotencyKey{
			Key:       scoped,
			UserID:    uid,
			Method:    c.Request.Method,
			Path:      c.Request.URL.Path,
			Status:    cw.Status(),
			Response:  body,
			CreatedAt: time.Now().UTC(),
		}); err != nil {
			s.logger.Warnw("idempotency save failed", "error", err)
		}
	}
}
