package middleware

import (
	"github.com/gin-gonic/gin"
)

// StreamingCORS returns middleware for streaming endpoints that need broader CORS access.
// Unlike the standard CORS middleware, this allows any origin for HLS segment loading
// while still setting proper credentials and method headers.
//
// This is used for endpoints that serve video segments (.m3u8, .ts, .mp4) which
// browsers fetch cross-origin via HLS.js.
func StreamingCORS() gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")

		// For streaming, allow any origin but reflect the specific origin
		// rather than using wildcard, which doesn't work with credentials
		if origin != "" {
			c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
		} else {
			c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		}

		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Authorization, Range, Origin")
		c.Writer.Header().Set("Access-Control-Expose-Headers", "Content-Length, Content-Range, X-Session-ID, X-Available-Qualities")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Max-Age", "86400")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}
