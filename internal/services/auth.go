package services

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/ngenohkevin/lms/internal/models"
	"github.com/redis/go-redis/v9"
	"golang.org/x/crypto/argon2"
)

var (
	ErrInvalidCredentials = errors.New("invalid username or password")
	ErrUserNotFound       = errors.New("user not found")
	ErrUserInactive       = errors.New("user account is inactive")
	ErrInvalidToken       = errors.New("invalid token")
	ErrTokenExpired       = errors.New("token has expired")
	ErrInvalidPassword    = errors.New("invalid password format")
	ErrInvalidRSAKey      = errors.New("invalid RSA key")
	ErrInvalidResetToken  = errors.New("invalid or expired reset token")
)

type AuthService struct {
	jwtPrivateKey     *rsa.PrivateKey
	jwtPublicKey      *rsa.PublicKey
	refreshPrivateKey *rsa.PrivateKey
	refreshPublicKey  *rsa.PublicKey
	tokenExpiry       time.Duration
	refreshExpiry     time.Duration
	argon2Config      *Argon2Config
	logger            *slog.Logger
	redisClient       *redis.Client
}

type Argon2Config struct {
	Memory      uint32
	Iterations  uint32
	Parallelism uint8
	SaltLength  uint32
	KeyLength   uint32
}

func NewAuthService(jwtPrivateKeyPEM, refreshPrivateKeyPEM string, tokenExpiry, refreshExpiry time.Duration, logger *slog.Logger, redisClient *redis.Client) (*AuthService, error) {
	// Parse JWT private key
	jwtPrivateKey, err := parseRSAPrivateKey(jwtPrivateKeyPEM)
	if err != nil {
		return nil, fmt.Errorf("failed to parse JWT private key: %w", err)
	}

	// Parse refresh private key
	refreshPrivateKey, err := parseRSAPrivateKey(refreshPrivateKeyPEM)
	if err != nil {
		return nil, fmt.Errorf("failed to parse refresh private key: %w", err)
	}

	return &AuthService{
		jwtPrivateKey:     jwtPrivateKey,
		jwtPublicKey:      &jwtPrivateKey.PublicKey,
		refreshPrivateKey: refreshPrivateKey,
		refreshPublicKey:  &refreshPrivateKey.PublicKey,
		tokenExpiry:       tokenExpiry,
		refreshExpiry:     refreshExpiry,
		argon2Config: &Argon2Config{
			Memory:      64 * 1024,
			Iterations:  3,
			Parallelism: 2,
			SaltLength:  16,
			KeyLength:   32,
		},
		logger:      logger,
		redisClient: redisClient,
	}, nil
}

func parseRSAPrivateKey(privateKeyPEM string) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode([]byte(privateKeyPEM))
	if block == nil {
		return nil, ErrInvalidRSAKey
	}

	privateKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		// Try PKCS8 format
		parsedKey, err := x509.ParsePKCS8PrivateKey(block.Bytes)
		if err != nil {
			return nil, err
		}
		if rsaKey, ok := parsedKey.(*rsa.PrivateKey); ok {
			return rsaKey, nil
		}
		return nil, ErrInvalidRSAKey
	}

	return privateKey, nil
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

	accessToken := jwt.NewWithClaims(jwt.SigningMethodRS256, accessClaims)
	accessTokenString, err := accessToken.SignedString(s.jwtPrivateKey)
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

	refreshToken := jwt.NewWithClaims(jwt.SigningMethodRS256, refreshClaims)
	refreshTokenString, err := refreshToken.SignedString(s.refreshPrivateKey)
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

	accessToken := jwt.NewWithClaims(jwt.SigningMethodRS256, accessClaims)
	accessTokenString, err := accessToken.SignedString(s.jwtPrivateKey)
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

	refreshToken := jwt.NewWithClaims(jwt.SigningMethodRS256, refreshClaims)
	refreshTokenString, err := refreshToken.SignedString(s.refreshPrivateKey)
	if err != nil {
		return "", "", err
	}

	return accessTokenString, refreshTokenString, nil
}

func (s *AuthService) ValidateToken(tokenString string) (*models.JWTClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &models.JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return s.jwtPublicKey, nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*models.JWTClaims); ok && token.Valid {
		// Check if token is blacklisted
		if s.redisClient != nil {
			ctx := context.Background()
			blacklisted, err := s.redisClient.Exists(ctx, fmt.Sprintf("blacklist:%s", tokenString)).Result()
			if err != nil {
				s.logger.Error("Failed to check token blacklist", "error", err)
				// Continue validation if Redis is down
			}
			if blacklisted > 0 {
				return nil, ErrInvalidToken
			}
		}
		return claims, nil
	}

	return nil, ErrInvalidToken
}

func (s *AuthService) ValidateRefreshToken(tokenString string) (*models.RefreshTokenClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &models.RefreshTokenClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return s.refreshPublicKey, nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*models.RefreshTokenClaims); ok && token.Valid {
		// Check if refresh token is blacklisted
		if s.redisClient != nil {
			ctx := context.Background()
			blacklisted, err := s.redisClient.Exists(ctx, fmt.Sprintf("blacklist:refresh:%s", tokenString)).Result()
			if err != nil {
				s.logger.Error("Failed to check refresh token blacklist", "error", err)
				// Continue validation if Redis is down
			}
			if blacklisted > 0 {
				return nil, ErrInvalidToken
			}
		}
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

// BlacklistToken adds a token to the blacklist
func (s *AuthService) BlacklistToken(tokenString string) error {
	if s.redisClient == nil {
		return errors.New("redis client not configured")
	}

	ctx := context.Background()

	// Parse token to get expiry time
	token, err := jwt.ParseWithClaims(tokenString, &models.JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return s.jwtPublicKey, nil
	})

	if err != nil {
		return err
	}

	if claims, ok := token.Claims.(*models.JWTClaims); ok {
		// Set blacklist entry with expiry time
		expiry := time.Until(claims.ExpiresAt.Time)
		if expiry <= 0 {
			// Token already expired, no need to blacklist
			return nil
		}

		err = s.redisClient.Set(ctx, fmt.Sprintf("blacklist:%s", tokenString), "1", expiry).Err()
		if err != nil {
			s.logger.Error("Failed to blacklist token", "error", err)
			return err
		}

		s.logger.Info("Token blacklisted successfully", "user_id", claims.UserID)
		return nil
	}

	return ErrInvalidToken
}

// BlacklistRefreshToken adds a refresh token to the blacklist
func (s *AuthService) BlacklistRefreshToken(tokenString string) error {
	if s.redisClient == nil {
		return errors.New("redis client not configured")
	}

	ctx := context.Background()

	// Parse token to get expiry time
	token, err := jwt.ParseWithClaims(tokenString, &models.RefreshTokenClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return s.refreshPublicKey, nil
	})

	if err != nil {
		return err
	}

	if claims, ok := token.Claims.(*models.RefreshTokenClaims); ok {
		// Set blacklist entry with expiry time
		expiry := time.Until(claims.ExpiresAt.Time)
		if expiry <= 0 {
			// Token already expired, no need to blacklist
			return nil
		}

		err = s.redisClient.Set(ctx, fmt.Sprintf("blacklist:refresh:%s", tokenString), "1", expiry).Err()
		if err != nil {
			s.logger.Error("Failed to blacklist refresh token", "error", err)
			return err
		}

		s.logger.Info("Refresh token blacklisted successfully", "user_id", claims.UserID)
		return nil
	}

	return ErrInvalidToken
}

// GeneratePasswordResetToken generates a secure password reset token
func (s *AuthService) GeneratePasswordResetToken(email string) (string, error) {
	if s.redisClient == nil {
		return "", errors.New("redis client not configured")
	}

	ctx := context.Background()

	// Generate a secure random token
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return "", fmt.Errorf("failed to generate token: %w", err)
	}

	token := base64.URLEncoding.EncodeToString(tokenBytes)

	// Store token in Redis with 1 hour expiry
	key := fmt.Sprintf("password_reset:%s", token)
	err := s.redisClient.Set(ctx, key, email, time.Hour).Err()
	if err != nil {
		s.logger.Error("Failed to store password reset token", "error", err)
		return "", err
	}

	s.logger.Info("Password reset token generated", "email", email)
	return token, nil
}

// ValidatePasswordResetToken validates a password reset token and returns the associated email
func (s *AuthService) ValidatePasswordResetToken(token string) (string, error) {
	if s.redisClient == nil {
		return "", errors.New("redis client not configured")
	}

	ctx := context.Background()
	key := fmt.Sprintf("password_reset:%s", token)

	email, err := s.redisClient.Get(ctx, key).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return "", ErrInvalidResetToken
		}
		s.logger.Error("Failed to validate password reset token", "error", err)
		return "", err
	}

	return email, nil
}

// InvalidatePasswordResetToken removes a password reset token from Redis
func (s *AuthService) InvalidatePasswordResetToken(token string) error {
	if s.redisClient == nil {
		return errors.New("redis client not configured")
	}

	ctx := context.Background()
	key := fmt.Sprintf("password_reset:%s", token)

	err := s.redisClient.Del(ctx, key).Err()
	if err != nil {
		s.logger.Error("Failed to invalidate password reset token", "error", err)
		return err
	}

	s.logger.Info("Password reset token invalidated")
	return nil
}
