package nfo

import (
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Writer handles writing NFO files to disk
type Writer struct{}

// NewWriter creates a new NFO writer
func NewWriter() *Writer {
	return &Writer{}
}

// WriteMovieNFO writes a movie NFO file to disk
// The NFO file path is derived from the media file path (same name, .nfo extension)
func (w *Writer) WriteMovieNFO(mediaPath string, metadata *MovieNFO) error {
	nfoPath := mediaPathToNFOPath(mediaPath)

	// Check if NFO already exists - don't overwrite user edits
	if fileExists(nfoPath) {
		return nil
	}

	return w.writeNFO(nfoPath, metadata)
}

// WriteTVShowNFO writes a tvshow.nfo file to disk
// The NFO is written to the show directory
func (w *Writer) WriteTVShowNFO(showDir string, metadata *TVShowNFO) error {
	nfoPath := filepath.Join(showDir, "tvshow.nfo")

	// Check if NFO already exists - don't overwrite user edits
	if fileExists(nfoPath) {
		return nil
	}

	return w.writeNFO(nfoPath, metadata)
}

// WriteEpisodeNFO writes an episode NFO file to disk
// The NFO file path is derived from the media file path (same name, .nfo extension)
func (w *Writer) WriteEpisodeNFO(mediaPath string, metadata *EpisodeNFO) error {
	nfoPath := mediaPathToNFOPath(mediaPath)

	// Check if NFO already exists - don't overwrite user edits
	if fileExists(nfoPath) {
		return nil
	}

	return w.writeNFO(nfoPath, metadata)
}

// writeNFO writes the NFO XML to disk
func (w *Writer) writeNFO(nfoPath string, v interface{}) error {
	// Ensure directory exists
	dir := filepath.Dir(nfoPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create NFO directory: %w", err)
	}

	// Marshal to XML
	data, err := xml.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal NFO: %w", err)
	}

	// Build final content with XML header
	var sb strings.Builder
	sb.WriteString("<?xml version=\"1.0\" encoding=\"UTF-8\" standalone=\"yes\"?>\n")
	sb.Write(data)
	sb.WriteString("\n")

	// Write to file
	if err := os.WriteFile(nfoPath, []byte(sb.String()), 0644); err != nil {
		return fmt.Errorf("failed to write NFO file: %w", err)
	}

	return nil
}

// mediaPathToNFOPath converts a media file path to an NFO file path
// Example: /movies/Inception.mkv -> /movies/Inception.nfo
func mediaPathToNFOPath(mediaPath string) string {
	dir := filepath.Dir(mediaPath)
	base := strings.TrimSuffix(filepath.Base(mediaPath), filepath.Ext(mediaPath))
	return filepath.Join(dir, base+".nfo")
}

// BuildMovieNFO creates a MovieNFO struct from metadata
// This is a convenience function for building the NFO from enrichment data
func BuildMovieNFO(
	title, originalTitle, plot, tagline string,
	year int,
	releaseDate string,
	runtimeMinutes int,
	director string,
	genres []string,
	imdbID, tmdbID string,
	contentRating string,
	studio string,
	country string,
	originalLanguage string,
) *MovieNFO {
	nfo := &MovieNFO{
		Title:            title,
		OriginalTitle:    originalTitle,
		SortTitle:        normalizeSortTitle(title),
		Year:             year,
		ReleaseDate:      releaseDate,
		Plot:             plot,
		Tagline:          tagline,
		Runtime:          fmt.Sprintf("%d", runtimeMinutes),
		Director:         director,
		Genres:           genres,
		IMDb:             imdbID,
		TMDbID:           tmdbID,
		MPAARating:       contentRating,
		Studio:           studio,
		Country:          country,
		OriginalLanguage: originalLanguage,
	}

	if runtimeMinutes > 0 {
		nfo.Runtime = fmt.Sprintf("%d", runtimeMinutes)
	}

	return nfo
}

// BuildEpisodeNFO creates an EpisodeNFO struct from metadata
func BuildEpisodeNFO(
	title, showTitle, plot string,
	season, episode int,
	airDate string,
	runtimeMinutes int,
	director string,
	imdbID, tmdbID string,
) *EpisodeNFO {
	nfo := &EpisodeNFO{
		Title:     title,
		ShowTitle: showTitle,
		Season:    season,
		Episode:   episode,
		Aired:     airDate,
		Plot:      plot,
		Director:  director,
		IMDb:      imdbID,
		TMDbID:    tmdbID,
	}

	if runtimeMinutes > 0 {
		nfo.Runtime = fmt.Sprintf("%d", runtimeMinutes)
	}

	return nfo
}

// normalizeSortTitle normalizes a title for consistent sorting
// Removes leading articles (The, A, An) and converts to lowercase
func normalizeSortTitle(title string) string {
	title = strings.TrimSpace(title)
	title = strings.ToLower(title)

	// Remove leading articles
	for _, article := range []string{"the ", "a ", "an "} {
		if strings.HasPrefix(title, article) {
			title = strings.TrimPrefix(title, article)
			break
		}
	}

	return strings.TrimSpace(title)
}
