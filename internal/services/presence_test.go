package services

import (
	"context"
	"log/slog"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var presenceTestLogger = slog.New(slog.NewTextHandler(os.Stdout, nil))

func TestPresenceService_NilRedis(t *testing.T) {
	// Presence service should gracefully handle nil Redis client
	service := NewPresenceService(nil, presenceTestLogger)

	ctx := context.Background()

	t.Run("UpdatePresence returns nil with nil redis", func(t *testing.T) {
		err := service.UpdatePresence(ctx, UserPresenceInfo{
			UserID:   1,
			Username: "testuser",
			Role:     "admin",
		})
		assert.NoError(t, err)
	})

	t.Run("GetOnlineUsers returns empty with nil redis", func(t *testing.T) {
		resp, err := service.GetOnlineUsers(ctx)
		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Empty(t, resp.Users)
		assert.Equal(t, 0, resp.Total)
	})

	t.Run("RemovePresence returns nil with nil redis", func(t *testing.T) {
		err := service.RemovePresence(ctx, 1)
		assert.NoError(t, err)
	})
}

func TestPresenceService_Interface(t *testing.T) {
	// Verify PresenceService implements PresenceServiceInterface
	var _ PresenceServiceInterface = (*PresenceService)(nil)
}

func TestParseDevice(t *testing.T) {
	tests := []struct {
		name     string
		ua       string
		expected string
	}{
		{"iPhone", "Mozilla/5.0 (iPhone; CPU iPhone OS 17_0 like Mac OS X)", "iphone"},
		{"iPad", "Mozilla/5.0 (iPad; CPU OS 17_0 like Mac OS X)", "ipad"},
		{"Android phone", "Mozilla/5.0 (Linux; Android 14; Pixel 8) Mobile", "android_phone"},
		{"Android tablet", "Mozilla/5.0 (Linux; Android 14; SM-X200)", "android_tablet"},
		{"Mac", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36", "mac"},
		{"Windows", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36", "windows"},
		{"Linux", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36", "linux"},
		{"Chromebook", "Mozilla/5.0 (X11; CrOS x86_64 14541.0.0)", "chromebook"},
		{"empty", "", "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseDevice(tt.ua)
			assert.Equal(t, tt.expected, result)
		})
	}
}
