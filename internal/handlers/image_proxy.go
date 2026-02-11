package handlers

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

var allowedImageDomains = map[string]bool{
	"books.google.com":       true,
	"covers.openlibrary.org": true,
}

const (
	imageCacheTTL    = 7 * 24 * time.Hour
	imageCachePrefix = "img:"
	maxImageSize     = 2 * 1024 * 1024 // 2MB
	fetchTimeout     = 10 * time.Second
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

	if !allowedImageDomains[parsed.Hostname()] {
		c.JSON(http.StatusForbidden, gin.H{"error": "domain not allowed"})
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

	// Fetch from external URL
	req, err := http.NewRequestWithContext(ctx, "GET", rawURL, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create request"})
		return
	}

	resp, err := h.httpClient.Do(req)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "failed to fetch image"})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		c.JSON(resp.StatusCode, gin.H{"error": fmt.Sprintf("upstream returned %d", resp.StatusCode)})
		return
	}

	limitedReader := io.LimitReader(resp.Body, maxImageSize+1)
	data, err := io.ReadAll(limitedReader)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "failed to read image"})
		return
	}
	if len(data) > maxImageSize {
		c.JSON(http.StatusBadRequest, gin.H{"error": "image too large"})
		return
	}

	contentType := resp.Header.Get("Content-Type")
	if contentType == "" || !strings.HasPrefix(contentType, "image/") {
		contentType = "image/jpeg"
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

// InvalidateCache removes a cached image by URL.
func (h *ImageProxyHandler) InvalidateCache(ctx context.Context, rawURL string) {
	if h.redis == nil || rawURL == "" {
		return
	}
	hash := sha256.Sum256([]byte(rawURL))
	key := imageCachePrefix + hex.EncodeToString(hash[:])
	h.redis.Del(ctx, key, key+":ct")
}
