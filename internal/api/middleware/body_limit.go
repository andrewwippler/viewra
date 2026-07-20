package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// BodyLimit returns middleware that limits request body size.
// This prevents memory exhaustion from oversized request bodies.
//
// maxSize is in bytes. Common values:
// - 1 << 20 (1 MB) for form data
// - 10 << 20 (10 MB) for file uploads
// - 100 << 20 (100 MB) for large payloads
func BodyLimit(maxSize int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.ContentLength > maxSize {
			c.JSON(http.StatusRequestEntityTooLarge, gin.H{
				"error": "request body too large",
			})
			c.Abort()
			return
		}

		// Also enforce via MaxBytesReader for streaming bodies
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxSize)

		c.Next()
	}
}

// DefaultBodyLimit returns a sensible default body size limit (10 MB).
func DefaultBodyLimit() int64 {
	return 10 << 20
}
