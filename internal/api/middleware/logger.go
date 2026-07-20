package middleware

import (
	"log/slog"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// Logger returns a middleware that logs HTTP requests using structured logging
func Logger(logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := sanitizeQueryForLogging(c.Request.URL.RawQuery)

		// Process request
		c.Next()

		// Calculate latency
		latency := time.Since(start)

		// Get status code
		status := c.Writer.Status()

		// Build log attributes
		attrs := []any{
			"method", c.Request.Method,
			"path", path,
			"status", status,
			"latency", latency.String(),
			"ip", c.ClientIP(),
			"user_agent", c.Request.UserAgent(),
		}

		// Add query string if present
		if query != "" {
			attrs = append(attrs, "query", query)
		}

		// Add error if present
		if len(c.Errors) > 0 {
			attrs = append(attrs, "errors", c.Errors.String())
		}

		// Log based on status code
		// Only log warnings (4xx) and errors (5xx) to reduce log pollution
		switch {
		case status >= 500:
			logger.Error("HTTP request", attrs...)
		case status >= 400:
			// HLS segment 404s are normal - player probes for next segment
			// Don't log them as warnings to avoid noise (supports both .ts and .m4s segments)
			isHLSSegment := strings.Contains(path, "/hls/") &&
				(strings.HasSuffix(path, ".ts") || strings.HasSuffix(path, ".m4s"))
			if status == 404 && isHLSSegment {
				logger.Debug("HTTP request", attrs...)
			} else {
				logger.Warn("HTTP request", attrs...)
			}
		// Successful requests (2xx, 3xx) are not logged
		}
	}
}

// sensitiveQueryParams are query parameters that may contain authentication tokens.
var sensitiveQueryParams = []string{"api_key", "ApiKey", "token", "access_token"}

// sanitizeQueryForLogging strips sensitive parameters from query strings before logging.
func sanitizeQueryForLogging(rawQuery string) string {
	if rawQuery == "" {
		return ""
	}

	params, err := url.ParseQuery(rawQuery)
	if err != nil {
		return ""
	}

	for _, key := range sensitiveQueryParams {
		if _, exists := params[key]; exists {
			params.Set(key, "[REDACTED]")
		}
	}

	return params.Encode()
}
