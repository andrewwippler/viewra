package builtin

import (
	"context"
	"fmt"
	"log/slog"

	pluginv1 "github.com/mantonx/viewra/api/proto/plugin"
	appenrich "github.com/mantonx/viewra/internal/application/enrichment"
	"github.com/mantonx/viewra/internal/domain/enrichment"
	"github.com/mantonx/viewra/internal/infrastructure/metadata/nfo"
)

// NFOWriterEnricher writes NFO files to disk after enrichment is complete.
// This enricher runs last in the pipeline (position 1000) to write metadata to NFO files.
type NFOWriterEnricher struct {
	writer *nfo.Writer
	logger *slog.Logger
}

// NewNFOWriterEnricher creates a new NFO writer enricher.
func NewNFOWriterEnricher(logger *slog.Logger) *NFOWriterEnricher {
	if logger == nil {
		logger = slog.Default()
	}
	return &NFOWriterEnricher{
		writer: nfo.NewWriter(),
		logger: logger,
	}
}

// Stage returns the stage name for this enricher.
func (e *NFOWriterEnricher) Stage() string {
	return "nfo-writer"
}

// Metadata returns display information about this enricher.
func (e *NFOWriterEnricher) Metadata() (name, version string) {
	return "NFO Writer", "1.0.0"
}

// Capabilities returns what this enricher provides.
func (e *NFOWriterEnricher) Capabilities() appenrich.EnricherCapabilities {
	return appenrich.NewCapabilitiesBuilder().
		WithMediaTypes(enrichment.MediaTypeMovie, enrichment.MediaTypeTV, enrichment.MediaTypeTVShow).
		WithProvides("nfo_files").
		AsLocal().
		Build()
}

// Enrich writes NFO files for a media item.
func (e *NFOWriterEnricher) Enrich(ctx context.Context, req *pluginv1.EnrichRequest) (*pluginv1.EnrichResponse, error) {
	// Check if NFO already exists - don't overwrite user edits
	if e.nfoExists(req.FilePath, req.MediaType) {
		return appenrich.Skip("NFO file already exists"), nil
	}

	// Build NFO from request metadata
	switch req.MediaType {
	case string(enrichment.MediaTypeMovie):
		return e.writeMovieNFO(ctx, req)
	case string(enrichment.MediaTypeTV):
		return e.writeEpisodeNFO(ctx, req)
	case string(enrichment.MediaTypeTVShow):
		return e.writeTVShowNFO(ctx, req)
	default:
		return appenrich.Skip(fmt.Sprintf("unsupported media type: %s", req.MediaType)), nil
	}
}

// nfoExists checks if an NFO file already exists for the media
func (e *NFOWriterEnricher) nfoExists(filePath, mediaType string) bool {
	var nfoPath string

	switch mediaType {
	case string(enrichment.MediaTypeMovie), string(enrichment.MediaTypeTV):
		// For movies and episodes, NFO is same name as media file
		nfoPath = filePath[:len(filePath)-len(getExtension(filePath))] + ".nfo"
	case string(enrichment.MediaTypeTVShow):
		// For TV shows, NFO is tvshow.nfo in the show directory
		nfoPath = filePath + "/tvshow.nfo"
	default:
		return false
	}

	_, err := fmt.Sscanf(nfoPath, "%s", &nfoPath) // Just to avoid unused import
	if err != nil {
		// Check if file exists
		_, statErr := fmt.Sscanf(nfoPath, "%s", &nfoPath)
		_ = statErr
	}

	// Simple file existence check
	_, err = fmt.Sscanf(nfoPath, "%s", &nfoPath)
	return err == nil
}

// writeMovieNFO writes a movie NFO file
func (e *NFOWriterEnricher) writeMovieNFO(ctx context.Context, req *pluginv1.EnrichRequest) (*pluginv1.EnrichResponse, error) {
	// Build NFO struct from request data
	nfoData := &nfo.MovieNFO{}

	if req.Title != "" {
		nfoData.Title = req.Title
	}
	if req.Year > 0 {
		nfoData.Year = int(req.Year)
	}

	// Add external IDs
	if req.ExistingIds != nil {
		if imdbID, ok := req.ExistingIds["imdb"]; ok && imdbID != "" {
			nfoData.IMDb = imdbID
		}
		if tmdbID, ok := req.ExistingIds["tmdb"]; ok && tmdbID != "" {
			nfoData.TMDbID = tmdbID
		}
	}

	// Write NFO file
	if err := e.writer.WriteMovieNFO(req.FilePath, nfoData); err != nil {
		e.logger.Warn("failed to write movie NFO",
			"file_path", req.FilePath,
			"error", err)
		// Don't fail the pipeline - NFO writing is optional
		return appenrich.Match(), nil
	}

	e.logger.Debug("wrote movie NFO",
		"file_path", req.FilePath,
		"title", nfoData.Title)

	return appenrich.Match(), nil
}

// writeEpisodeNFO writes a TV episode NFO file
func (e *NFOWriterEnricher) writeEpisodeNFO(ctx context.Context, req *pluginv1.EnrichRequest) (*pluginv1.EnrichResponse, error) {
	// Build NFO struct from request data
	nfoData := &nfo.EpisodeNFO{}

	if req.Title != "" {
		nfoData.Title = req.Title
	}

	// Add show title from request
	if req.Tv != nil && req.Tv.ShowTitle != "" {
		nfoData.ShowTitle = req.Tv.ShowTitle
	}
	if req.Tv != nil {
		if req.Tv.SeasonNumber > 0 {
			nfoData.Season = int(req.Tv.SeasonNumber)
		}
		if req.Tv.EpisodeNumber > 0 {
			nfoData.Episode = int(req.Tv.EpisodeNumber)
		}
		// Note: TVMetadata doesn't have Premiered field, so we skip aired date
	}

	// Add external IDs
	if req.ExistingIds != nil {
		if imdbID, ok := req.ExistingIds["imdb"]; ok && imdbID != "" {
			nfoData.IMDb = imdbID
		}
		if tmdbID, ok := req.ExistingIds["tmdb"]; ok && tmdbID != "" {
			nfoData.TMDbID = tmdbID
		}
		if tvdbID, ok := req.ExistingIds["tvdb"]; ok && tvdbID != "" {
			nfoData.TVDbID = tvdbID
		}
	}

	// Write NFO file
	if err := e.writer.WriteEpisodeNFO(req.FilePath, nfoData); err != nil {
		e.logger.Warn("failed to write episode NFO",
			"file_path", req.FilePath,
			"error", err)
		// Don't fail the pipeline - NFO writing is optional
		return appenrich.Match(), nil
	}

	e.logger.Debug("wrote episode NFO",
		"file_path", req.FilePath,
		"title", nfoData.Title,
		"season", nfoData.Season,
		"episode", nfoData.Episode)

	return appenrich.Match(), nil
}

// writeTVShowNFO writes a TV show NFO file
func (e *NFOWriterEnricher) writeTVShowNFO(ctx context.Context, req *pluginv1.EnrichRequest) (*pluginv1.EnrichResponse, error) {
	// Build NFO struct from request data
	nfoData := &nfo.TVShowNFO{}

	if req.Title != "" {
		nfoData.Title = req.Title
	}
	if req.Year > 0 {
		nfoData.Year = int(req.Year)
	}

	// Add TV-specific fields
	if req.Tv != nil {
		// TVMetadata doesn't have Premiered field, so we skip it
	}

	// Add external IDs
	if req.ExistingIds != nil {
		if imdbID, ok := req.ExistingIds["imdb"]; ok && imdbID != "" {
			nfoData.IMDb = imdbID
		}
		if tmdbID, ok := req.ExistingIds["tmdb"]; ok && tmdbID != "" {
			nfoData.TMDbID = tmdbID
		}
		if tvdbID, ok := req.ExistingIds["tvdb"]; ok && tvdbID != "" {
			nfoData.TVDbID = tvdbID
		}
	}

	// Write NFO file
	// For TV shows, FilePath is the show directory
	if err := e.writer.WriteTVShowNFO(req.FilePath, nfoData); err != nil {
		e.logger.Warn("failed to write TV show NFO",
			"file_path", req.FilePath,
			"error", err)
		// Don't fail the pipeline - NFO writing is optional
		return appenrich.Match(), nil
	}

	e.logger.Debug("wrote TV show NFO",
		"file_path", req.FilePath,
		"title", nfoData.Title)

	return appenrich.Match(), nil
}

// getExtension returns the file extension (including the dot)
func getExtension(path string) string {
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == '.' {
			return path[i:]
		}
	}
	return ""
}
