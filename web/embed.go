//go:build !dev
// +build !dev

package web

import (
	"embed"
	"io/fs"
)

// DistFS embeds the built frontend files for production.
// Both web and TV frontend builds are embedded simultaneously.
// Web frontend is served at /, TV frontend is served at /tv/.
// This only happens when building WITHOUT -tags dev.
//
//go:embed dist
var embedFS embed.FS

// WebFS returns the web frontend filesystem (served at /)
func WebFS() (fs.FS, error) {
	return fs.Sub(embedFS, "dist/web")
}

// TvFS returns the TV frontend filesystem (served at /tv/)
func TvFS() (fs.FS, error) {
	return fs.Sub(embedFS, "dist/tv")
}

// FS returns the web frontend filesystem for backward compatibility
func FS() (fs.FS, error) {
	return WebFS()
}

// IsEmbedded returns true when frontend is embedded
func IsEmbedded() bool {
	return true
}
