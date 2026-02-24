package handlers

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

const (
	imageCacheTTL    = 7 * 24 * time.Hour
	imageCachePrefix = "img:"
	maxImageSize     = 2 * 1024 * 1024 // 2MB
	fetchTimeout     = 10 * time.Second
	maxRetries       = 3
)

type ImageProxyHandler struct {
	redis      *redis.Client
	httpClient *http.Client
}

func NewImageProxyHandler(rc *redis.Client) *ImageProxyHandler {
	return &ImageProxyHandler{
		redis: rc,
		httpClient: &http.Client{
			Timeout: fetchTimeout,
		},
	}
}

func (h *ImageProxyHandler) ProxyImage(c *gin.Context) {
	rawURL := c.Query("url")
	if rawURL == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "url parameter is required"})
		return
	}

	parsed, err := url.Parse(rawURL)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid URL"})
		return
	}

	// Block private/internal IPs to prevent SSRF
	if isPrivateHost(parsed.Hostname()) {
		c.JSON(http.StatusForbidden, gin.H{"error": "internal addresses not allowed"})
		return
	}

	hash := sha256.Sum256([]byte(rawURL))
	key := imageCachePrefix + hex.EncodeToString(hash[:])
	contentTypeKey := key + ":ct"

	ctx := c.Request.Context()

	// Check Redis cache
	if h.redis != nil {
		cached, err := h.redis.Get(ctx, key).Bytes()
		if err == nil {
			contentType, _ := h.redis.Get(ctx, contentTypeKey).Result()
			if contentType == "" {
				contentType = "image/jpeg"
			}
			c.Header("X-Cache", "HIT")
			c.Header("Cache-Control", "public, max-age=604800")
			c.Data(http.StatusOK, contentType, cached)
			return
		}
	}

	// Fetch from external URL with retry on 429
	data, contentType, err := h.fetchWithRetry(ctx, rawURL)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "failed to fetch image"})
		return
	}

	// Cache in Redis
	if h.redis != nil {
		cacheCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		h.redis.Set(cacheCtx, key, data, imageCacheTTL)
		h.redis.Set(cacheCtx, contentTypeKey, contentType, imageCacheTTL)
	}

	c.Header("X-Cache", "MISS")
	c.Header("Cache-Control", "public, max-age=604800")
	c.Data(http.StatusOK, contentType, data)
}

// fetchWithRetry fetches an image, retrying with exponential backoff on 429.
func (h *ImageProxyHandler) fetchWithRetry(ctx context.Context, rawURL string) ([]byte, string, error) {
	var lastErr error

	for attempt := range maxRetries {
		if attempt > 0 {
			backoff := time.Duration(1<<uint(attempt-1)) * time.Second // 1s, 2s, 4s
			select {
			case <-ctx.Done():
				return nil, "", ctx.Err()
			case <-time.After(backoff):
			}
		}

		req, err := http.NewRequestWithContext(ctx, "GET", rawURL, nil)
		if err != nil {
			return nil, "", err
		}

		resp, err := h.httpClient.Do(req)
		if err != nil {
			lastErr = err
			continue
		}

		if resp.StatusCode == http.StatusTooManyRequests {
			resp.Body.Close()
			lastErr = http.ErrHandlerTimeout
			continue
		}

		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			lastErr = http.ErrNotSupported
			continue
		}

		limitedReader := io.LimitReader(resp.Body, maxImageSize+1)
		data, err := io.ReadAll(limitedReader)
		if err != nil {
			return nil, "", err
		}
		if len(data) > maxImageSize {
			return nil, "", http.ErrContentLength
		}

		contentType := resp.Header.Get("Content-Type")
		if contentType == "" || !strings.HasPrefix(contentType, "image/") {
			contentType = "image/jpeg"
		}

		return data, contentType, nil
	}

	return nil, "", lastErr
}

// InvalidateCache removes a cached image by URL.
func (h *ImageProxyHandler) InvalidateCache(ctx context.Context, rawURL string) {
	if h.redis == nil || rawURL == "" {
		return
	}
	hash := sha256.Sum256([]byte(rawURL))
	key := imageCachePrefix + hex.EncodeToString(hash[:])
	h.redis.Del(ctx, key, key+":ct")
}

// isPrivateHost checks if a hostname resolves to a private/internal IP.
func isPrivateHost(host string) bool {
	// Block localhost
	if host == "localhost" || host == "127.0.0.1" || host == "::1" {
		return true
	}

	// Resolve hostname and check if any IP is private
	ips, err := net.LookupIP(host)
	if err != nil {
		return false
	}
	for _, ip := range ips {
		if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
			return true
		}
	}
	return false
}
