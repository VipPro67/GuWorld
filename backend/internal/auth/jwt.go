package auth

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/pkg/logger"
)

var secretKey []byte

func init() {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		logger.S.Fatal("JWT_SECRET environment variable is required for secure token signing. Please set it before starting the server.")
	}
	if len(secret) < 32 {
		logger.S.Fatal("JWT_SECRET must be at least 32 characters long for adequate security.")
	}
	secretKey = []byte(secret)
}

// CustomClaims represents the JWT claims
type CustomClaims struct {
	PlayerID    string `json:"player_id"`
	Username    string `json:"username"`
	SessionID   string `json:"session_id"`
	jwt.RegisteredClaims
}

// GenerateToken issues a new JWT token valid for 7 days
func GenerateToken(playerID uuid.UUID, username, sessionID string) (string, error) {
	claims := CustomClaims{
		PlayerID:  playerID.String(),
		Username:  username,
		SessionID: sessionID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * 7 * time.Hour)), // 7 days
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    "GuWorld",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(secretKey)
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}

	return tokenString, nil
}

// ValidateToken parses and validates the JWT token
func ValidateToken(tokenString string) (*CustomClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &CustomClaims{}, func(t *jwt.Token) (interface{}, error) {
		// Ensure token signing method is HS256
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return secretKey, nil
	})

	if err != nil {
		return nil, fmt.Errorf("invalid token: %w", err)
	}

	if claims, ok := token.Claims.(*CustomClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, errors.New("invalid token claims")
}
