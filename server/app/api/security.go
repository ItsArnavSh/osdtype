package api

import (
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
)

const maxRequestBodyBytes int64 = 1 << 20 // 1 MiB

// validateProductionConfig refuses to start with the development JWT secret
// when the server is running in release mode or with developer login disabled.
// Cookie settings that browsers would silently reject are also fatal.
func validateProductionConfig() error {
	release := strings.EqualFold(strings.TrimSpace(os.Getenv("GIN_MODE")), "release")
	devLogin := viper.GetBool("Core.dev_login")
	if !release && devLogin {
		return nil
	}
	key := strings.TrimSpace(viper.GetString("Core.jwt_key"))
	if key == "" || key == "dev-insecure-change-me" {
		return fmt.Errorf("JWTKEY must be set to a strong secret when DEV_LOGIN=false or GIN_MODE=release")
	}
	if len(key) < 32 {
		return fmt.Errorf("JWTKEY must be at least 32 characters")
	}
	if strings.EqualFold(strings.TrimSpace(viper.GetString("Core.cookie_samesite")), "none") &&
		!viper.GetBool("Core.cookie_secure") {
		return fmt.Errorf("COOKIE_SECURE must be true when COOKIE_SAMESITE=none")
	}
	return nil
}

// securityHeaders applies conservative defaults to every API response. The
// static frontend sets its own headers in app/static/_headers.
func securityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		h := c.Writer.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("X-Frame-Options", "DENY")
		h.Set("Referrer-Policy", "strict-origin-when-cross-origin")
		h.Set("Permissions-Policy", "camera=(), microphone=(), geolocation=(), payment=(), usb=()")
		h.Set("Cross-Origin-Opener-Policy", "same-origin")
		h.Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		h.Set("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'; base-uri 'none'")
		c.Next()
	}
}

// bodyLimit rejects oversized request bodies before handlers allocate or parse
// them. Chunked bodies are capped by MaxBytesReader as well.
func bodyLimit(maxBytes int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.ContentLength > maxBytes {
			c.AbortWithStatusJSON(http.StatusRequestEntityTooLarge, apiError{
				Error:     "request body too large",
				Code:      "payload_too_large",
				RequestID: requestID(c),
			})
			return
		}
		if c.Request.Body != nil {
			c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBytes)
		}
		c.Next()
	}
}
