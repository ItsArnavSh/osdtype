package auth

import (
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/spf13/viper"
)

func jwtKey() []byte {
	key := viper.GetString("Core.jwt_key")
	if key == "" {
		key = os.Getenv("JWTKEY")
	}
	if key == "" {
		key = "dev-insecure-change-me"
	}
	return []byte(key)
}

// GenerateJWT creates a new token for a user id.
func GenerateJWT(userID string) (string, error) {
	claims := jwt.RegisteredClaims{
		Subject:   userID,
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(7 * 24 * time.Hour)),
		IssuedAt:  jwt.NewNumericDate(time.Now()),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtKey())
}

func ValidateJWT(tokenString string) (string, error) {
	token, err := jwt.ParseWithClaims(tokenString, &jwt.RegisteredClaims{}, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return jwtKey(), nil
	})
	if err != nil {
		return "", fmt.Errorf("token parsing failed: %w", err)
	}
	if claims, ok := token.Claims.(*jwt.RegisteredClaims); ok && token.Valid {
		return claims.Subject, nil
	}
	return "", fmt.Errorf("invalid token")
}

func authCookieSameSite() http.SameSite {
	switch strings.ToLower(strings.TrimSpace(viper.GetString("Core.cookie_samesite"))) {
	case "none":
		return http.SameSiteNoneMode
	case "strict":
		return http.SameSiteStrictMode
	default:
		return http.SameSiteLaxMode
	}
}

func SetAuthCookie(w http.ResponseWriter, token string) {
	secure := viper.GetBool("Core.cookie_secure")
	sameSite := authCookieSameSite()
	if sameSite == http.SameSiteNoneMode && !secure {
		// Browsers reject SameSite=None without Secure. Fall back rather than
		// silently dropping the session cookie in local HTTP development.
		sameSite = http.SameSiteLaxMode
	}
	http.SetCookie(w, &http.Cookie{
		Name:     "token",
		Value:    token,
		Path:     "/",
		MaxAge:   7 * 24 * 3600,
		HttpOnly: true,
		Secure:   secure,
		SameSite: sameSite,
	})
}

func ClearAuthCookie(w http.ResponseWriter) {
	secure := viper.GetBool("Core.cookie_secure")
	sameSite := authCookieSameSite()
	if sameSite == http.SameSiteNoneMode && !secure {
		sameSite = http.SameSiteLaxMode
	}
	http.SetCookie(w, &http.Cookie{
		Name:     "token",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   secure,
		SameSite: sameSite,
	})
}
