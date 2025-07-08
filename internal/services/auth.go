package services

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/ngenohkevin/lms/internal/models"
	"golang.org/x/crypto/argon2"
)

var (
	ErrInvalidCredentials = errors.New("invalid username or password")
	ErrUserNotFound       = errors.New("user not found")
	ErrUserInactive       = errors.New("user account is inactive")
	ErrInvalidToken       = errors.New("invalid token")
	ErrTokenExpired       = errors.New("token has expired")
	ErrInvalidPassword    = errors.New("invalid password format")
)

type AuthService struct {
	jwtSecret     []byte
	refreshSecret []byte
	tokenExpiry   time.Duration
	refreshExpiry time.Duration
	argon2Config  *Argon2Config
	logger        *slog.Logger
}

type Argon2Config struct {
	Memory      uint32
	Iterations  uint32
	Parallelism uint8
	SaltLength  uint32
	KeyLength   uint32
}

func NewAuthService(jwtSecret, refreshSecret []byte, tokenExpiry, refreshExpiry time.Duration, logger *slog.Logger) *AuthService {
	return &AuthService{
		jwtSecret:     jwtSecret,
		refreshSecret: refreshSecret,
		tokenExpiry:   tokenExpiry,
		refreshExpiry: refreshExpiry,
		argon2Config: &Argon2Config{
			Memory:      64 * 1024,
			Iterations:  3,
			Parallelism: 2,
			SaltLength:  16,
			KeyLength:   32,
		},
		logger: logger,
	}
}

func (s *AuthService) HashPassword(password string) (string, error) {
	if len(password) < 8 {
		return "", ErrInvalidPassword
	}

	salt := make([]byte, s.argon2Config.SaltLength)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}

	hash := argon2.IDKey(
		[]byte(password),
		salt,
		s.argon2Config.Iterations,
		s.argon2Config.Memory,
		s.argon2Config.Parallelism,
		s.argon2Config.KeyLength,
	)

	b64Salt := base64.RawStdEncoding.EncodeToString(salt)
	b64Hash := base64.RawStdEncoding.EncodeToString(hash)

	return fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version,
		s.argon2Config.Memory,
		s.argon2Config.Iterations,
		s.argon2Config.Parallelism,
		b64Salt,
		b64Hash,
	), nil
}

func (s *AuthService) VerifyPassword(hashedPassword, password string) (bool, error) {
	// Split the hash by $ delimiter
	parts := strings.Split(hashedPassword, "$")
	if len(parts) != 6 {
		return false, errors.New("invalid hash format")
	}

	if parts[1] != "argon2id" {
		return false, errors.New("invalid hash type")
	}

	var version int
	_, err := fmt.Sscanf(parts[2], "v=%d", &version)
	if err != nil {
		return false, fmt.Errorf("invalid version: %w", err)
	}

	if version != argon2.Version {
		return false, errors.New("incompatible argon2 version")
	}

	var memory, iterations uint32
	var parallelism uint8
	_, err = fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &memory, &iterations, &parallelism)
	if err != nil {
		return false, fmt.Errorf("invalid parameters: %w", err)
	}

	decodedSalt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return false, fmt.Errorf("error decoding salt: %w", err)
	}

	decodedHash, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return false, fmt.Errorf("error decoding hash: %w", err)
	}

	computedHash := argon2.IDKey(
		[]byte(password),
		decodedSalt,
		iterations,
		memory,
		parallelism,
		uint32(len(decodedHash)),
	)

	// Use constant time comparison
	if len(decodedHash) != len(computedHash) {
		return false, nil
	}

	for i := 0; i < len(decodedHash); i++ {
		if decodedHash[i] != computedHash[i] {
			return false, nil
		}
	}

	return true, nil
}

func (s *AuthService) GenerateTokens(user *models.User, userType string) (string, string, error) {
	now := time.Now()

	// Generate access token
	accessClaims := &models.JWTClaims{
		UserID:   user.ID,
		Username: user.Username,
		Role:     user.Role,
		UserType: userType,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(s.tokenExpiry)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			Subject:   fmt.Sprintf("user_%d", user.ID),
		},
	}

	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
	accessTokenString, err := accessToken.SignedString(s.jwtSecret)
	if err != nil {
		return "", "", err
	}

	// Generate refresh token
	refreshClaims := &models.RefreshTokenClaims{
		UserID:   user.ID,
		Username: user.Username,
		UserType: userType,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(s.refreshExpiry)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			Subject:   fmt.Sprintf("user_%d", user.ID),
		},
	}

	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims)
	refreshTokenString, err := refreshToken.SignedString(s.refreshSecret)
	if err != nil {
		return "", "", err
	}

	return accessTokenString, refreshTokenString, nil
}

func (s *AuthService) GenerateStudentTokens(student *models.Student) (string, string, error) {
	now := time.Now()

	// Generate access token
	accessClaims := &models.JWTClaims{
		UserID:   student.ID,
		Username: student.StudentID,
		Role:     "student",
		UserType: "student",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(s.tokenExpiry)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			Subject:   fmt.Sprintf("student_%d", student.ID),
		},
	}

	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
	accessTokenString, err := accessToken.SignedString(s.jwtSecret)
	if err != nil {
		return "", "", err
	}

	// Generate refresh token
	refreshClaims := &models.RefreshTokenClaims{
		UserID:   student.ID,
		Username: student.StudentID,
		UserType: "student",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(s.refreshExpiry)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			Subject:   fmt.Sprintf("student_%d", student.ID),
		},
	}

	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims)
	refreshTokenString, err := refreshToken.SignedString(s.refreshSecret)
	if err != nil {
		return "", "", err
	}

	return accessTokenString, refreshTokenString, nil
}

func (s *AuthService) ValidateToken(tokenString string) (*models.JWTClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &models.JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return s.jwtSecret, nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*models.JWTClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, ErrInvalidToken
}

func (s *AuthService) ValidateRefreshToken(tokenString string) (*models.RefreshTokenClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &models.RefreshTokenClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return s.refreshSecret, nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*models.RefreshTokenClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, ErrInvalidToken
}

func (s *AuthService) RefreshTokens(refreshTokenString string) (string, string, error) {
	claims, err := s.ValidateRefreshToken(refreshTokenString)
	if err != nil {
		return "", "", err
	}

	// Here we would typically get the user from the database
	// For now, we'll create a minimal user object
	user := &models.User{
		ID:       claims.UserID,
		Username: claims.Username,
		Role:     models.UserRole("librarian"), // Default role, should be fetched from DB
	}

	return s.GenerateTokens(user, claims.UserType)
}
