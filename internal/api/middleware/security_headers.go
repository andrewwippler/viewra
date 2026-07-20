package middleware

import (
	"github.com/gin-gonic/gin"
)

// SecurityHeadersConfig holds configuration for security headers middleware.
type SecurityHeadersConfig struct {
	// HSTSSeconds is the max-age value for Strict-Transport-Security.
	// Set to 0 to disable HSTS header.
	HSTSSeconds int

	// ContentSecurityPolicy is the CSP header value.
	ContentSecurityPolicy string

	// XContentTypeOptions sets X-Content-Type-Options.
	// Defaults to "nosniff" if empty.
	XContentTypeOptions string
}

// DefaultSecurityHeadersConfig returns a sensible default configuration.
func DefaultSecurityHeadersConfig() SecurityHeadersConfig {
	return SecurityHeadersConfig{
		HSTSSeconds:           31536000, // 1 year
		ContentSecurityPolicy: "default-src 'self'; script-src 'self' 'unsafe-inline' 'unsafe-eval'; style-src 'self' 'unsafe-inline'; img-src 'self' data: blob:; media-src 'self' blob:; connect-src 'self'; font-src 'self'; object-src 'none'",
		XContentTypeOptions:   "nosniff",
	}
}

// SecurityHeaders returns middleware that sets common security headers.
func SecurityHeaders(config SecurityHeadersConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		// HSTS - only over HTTPS (or when behind a proxy that terminates TLS)
		if config.HSTSSeconds > 0 {
			c.Writer.Header().Set("Strict-Transport-Security",
				"max-age="+itoa(config.HSTSSeconds)+"; includeSubDomains")
		}

		// Prevent MIME type sniffing
		if config.XContentTypeOptions != "" {
			c.Writer.Header().Set("X-Content-Type-Options", config.XContentTypeOptions)
		}

		// Content Security Policy
		if config.ContentSecurityPolicy != "" {
			c.Writer.Header().Set("Content-Security-Policy", config.ContentSecurityPolicy)
		}

		// Referrer policy - prevent leaking auth tokens in referrer headers
		c.Writer.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")

		// Permissions policy - disable unnecessary browser features
		c.Writer.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")

		c.Next()
	}
}

// itoa is a minimal int-to-string conversion to avoid importing strconv.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	buf := [20]byte{}
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}
