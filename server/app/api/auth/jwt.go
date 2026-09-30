package auth

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// signingKey returns the HMAC key used for both signing and validation.
//
// The key is read from the environment on each call rather than captured in a
// package-level variable at init time. Reading it lazily means the value is
// picked up whenever the process sets it, which is what tests and any caller
// that configures the environment after startup need.
func signingKey() []byte {
	return []byte(os.Getenv("JWTKEY"))
}

// GenerateJWT creates a new JWT for a given userID.
// It stores the userID in the 'Subject' claim.
func GenerateJWT(userID string) (string, error) {
	key := signingKey()
	if len(key) == 0 {
		return "", errors.New("JWTKEY environment variable not set")
	}

	claims := jwt.RegisteredClaims{
		Subject:   userID,
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
		IssuedAt:  jwt.NewNumericDate(time.Now()),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(key)
}

// ValidateJWT parses a token string and returns the userID (from the Subject claim).
// It returns an error if the token is invalid, expired, or malformed.
func ValidateJWT(tokenString string) (string, error) {
	key := signingKey()
	if len(key) == 0 {
		return "", errors.New("JWTKEY environment variable not set")
	}

	// Parse the token with the RegisteredClaims structure
	token, err := jwt.ParseWithClaims(tokenString, &jwt.RegisteredClaims{}, func(token *jwt.Token) (any, error) {
		// Check the signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		// Return the secret key for validation
		return key, nil
	})

	if err != nil {
		// This will handle errors like malformed tokens or signature mismatch
		return "", fmt.Errorf("token parsing failed: %w", err)
	}

	// Validate the token and extract the claims
	if claims, ok := token.Claims.(*jwt.RegisteredClaims); ok && token.Valid {
		// A token with no subject is not usable: every handler resolves the
		// subject to a numeric user id, and an empty one would silently become
		// user 0. Refuse it here so the failure is explicit.
		if claims.Subject == "" {
			return "", errors.New("token has no subject")
		}
		return claims.Subject, nil
	}

	return "", errors.New("invalid token")
}
