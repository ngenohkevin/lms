package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	presenceKeyPrefix = "presence:user:"
	presenceSetKey    = "presence:online_users"
	presenceTTL       = 15 * time.Minute
)

// UserPresenceInfo represents a user's presence data
type UserPresenceInfo struct {
	UserID    int       `json:"user_id"`
	Username  string    `json:"username"`
	Role      string    `json:"role"`
	LastSeen  time.Time `json:"last_seen"`
	IPAddr    string    `json:"ip_address,omitempty"`
	Path      string    `json:"path,omitempty"`
	UserAgent string    `json:"user_agent,omitempty"`
	Device    string    `json:"device,omitempty"`
}

// OnlineUsersResponse represents the response for online users endpoint
type OnlineUsersResponse struct {
	Users []UserPresenceInfo `json:"users"`
	Total int                `json:"total"`
}

// PresenceServiceInterface defines the interface for presence operations
type PresenceServiceInterface interface {
	UpdatePresence(ctx context.Context, info UserPresenceInfo) error
	GetOnlineUsers(ctx context.Context) (*OnlineUsersResponse, error)
	RemovePresence(ctx context.Context, userID int) error
}

// PresenceService handles user online presence tracking via Redis
type PresenceService struct {
	redisClient *redis.Client
	logger      *slog.Logger
}

// NewPresenceService creates a new PresenceService
func NewPresenceService(redisClient *redis.Client, logger *slog.Logger) *PresenceService {
	return &PresenceService{
		redisClient: redisClient,
		logger:      logger,
	}
}

// parseDevice extracts a device type from a User-Agent string
func parseDevice(ua string) string {
	ua = strings.ToLower(ua)

	switch {
	case strings.Contains(ua, "iphone"):
		return "iphone"
	case strings.Contains(ua, "ipad"):
		return "ipad"
	case strings.Contains(ua, "android") && strings.Contains(ua, "mobile"):
		return "android_phone"
	case strings.Contains(ua, "android"):
		return "android_tablet"
	case strings.Contains(ua, "macintosh") || strings.Contains(ua, "mac os"):
		return "mac"
	case strings.Contains(ua, "windows"):
		return "windows"
	case strings.Contains(ua, "linux"):
		return "linux"
	case strings.Contains(ua, "cros"):
		return "chromebook"
	default:
		return "unknown"
	}
}

// UpdatePresence updates a user's presence in Redis
func (s *PresenceService) UpdatePresence(ctx context.Context, info UserPresenceInfo) error {
	if s.redisClient == nil {
		return nil
	}

	info.LastSeen = time.Now()
	if info.UserAgent != "" && info.Device == "" {
		info.Device = parseDevice(info.UserAgent)
	}

	data, err := json.Marshal(info)
	if err != nil {
		return fmt.Errorf("failed to marshal presence info: %w", err)
	}

	key := fmt.Sprintf("%s%d", presenceKeyPrefix, info.UserID)

	pipe := s.redisClient.Pipeline()
	pipe.Set(ctx, key, data, presenceTTL)
	pipe.SAdd(ctx, presenceSetKey, info.UserID)
	_, err = pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to update presence: %w", err)
	}

	return nil
}

// GetOnlineUsers returns all currently online users
func (s *PresenceService) GetOnlineUsers(ctx context.Context) (*OnlineUsersResponse, error) {
	if s.redisClient == nil {
		return &OnlineUsersResponse{Users: []UserPresenceInfo{}, Total: 0}, nil
	}

	// Get all user IDs from the online set
	memberIDs, err := s.redisClient.SMembers(ctx, presenceSetKey).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get online user IDs: %w", err)
	}

	users := make([]UserPresenceInfo, 0, len(memberIDs))
	expiredIDs := make([]interface{}, 0)

	for _, idStr := range memberIDs {
		key := presenceKeyPrefix + idStr
		data, err := s.redisClient.Get(ctx, key).Result()
		if err == redis.Nil {
			// Key expired, clean up from set
			expiredIDs = append(expiredIDs, idStr)
			continue
		}
		if err != nil {
			s.logger.Warn("Failed to get presence data", "user_id", idStr, "error", err)
			continue
		}

		var info UserPresenceInfo
		if err := json.Unmarshal([]byte(data), &info); err != nil {
			s.logger.Warn("Failed to unmarshal presence data", "user_id", idStr, "error", err)
			continue
		}

		users = append(users, info)
	}

	// Clean up expired entries
	if len(expiredIDs) > 0 {
		if err := s.redisClient.SRem(ctx, presenceSetKey, expiredIDs...).Err(); err != nil {
			s.logger.Warn("Failed to clean up expired presence entries", "error", err)
		}
	}

	return &OnlineUsersResponse{
		Users: users,
		Total: len(users),
	}, nil
}

// RemovePresence removes a user's presence (called on logout)
func (s *PresenceService) RemovePresence(ctx context.Context, userID int) error {
	if s.redisClient == nil {
		return nil
	}

	key := fmt.Sprintf("%s%d", presenceKeyPrefix, userID)

	pipe := s.redisClient.Pipeline()
	pipe.Del(ctx, key)
	pipe.SRem(ctx, presenceSetKey, userID)
	_, err := pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to remove presence: %w", err)
	}

	return nil
}
