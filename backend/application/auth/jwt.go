package auth

import (
	"fmt"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// jwtKey is shared between generation and validation
var jwtKey = []byte(os.Getenv("JWTKEY"))

// GenerateJWT creates a new JWT for a given userID.
// It stores the userID in the 'Subject' claim.
func GenerateJWT(userID string) (string, error) {
	if len(jwtKey) == 0 {
		fmt.Println("Key Not Found")
		return "", fmt.Errorf("JWTKEY environment variable not set")
	}

	claims := jwt.RegisteredClaims{
		Subject:   userID,
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
		IssuedAt:  jwt.NewNumericDate(time.Now()),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtKey)
}

// ValidateJWT parses a token string and returns the userID (from the Subject claim).
// It returns an error if the token is invalid, expired, or malformed.
func ValidateJWT(tokenString string) (string, error) {
	if len(jwtKey) == 0 {
		return "", fmt.Errorf("JWTKEY environment variable not set")
	}

	// Parse the token with the RegisteredClaims structure
	token, err := jwt.ParseWithClaims(tokenString, &jwt.RegisteredClaims{}, func(token *jwt.Token) (any, error) {
		// Check the signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		// Return the secret key for validation
		return jwtKey, nil
	})

	if err != nil {
		// This will handle errors like malformed tokens or signature mismatch
		return "", fmt.Errorf("token parsing failed: %w", err)
	}

	// Validate the token and extract the claims
	if claims, ok := token.Claims.(*jwt.RegisteredClaims); ok && token.Valid {
		// The token is valid, return the Subject (which contains the userID)
		return claims.Subject, nil
	}

	return "", fmt.Errorf("invalid token")
}
