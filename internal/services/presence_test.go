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
