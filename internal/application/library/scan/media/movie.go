package media

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/mantonx/viewra/internal/application/enrichment/pipeline"
	"github.com/mantonx/viewra/internal/application/library/scan"
	"github.com/mantonx/viewra/internal/application/library/scan/scanutil"
	domainCommon "github.com/mantonx/viewra/internal/domain/common"
	"github.com/mantonx/viewra/internal/domain/enrichment"
	"github.com/mantonx/viewra/internal/domain/media"
	"github.com/mantonx/viewra/internal/domain/scanner"
)

// ProcessMovie creates or updates a movie entry.
// If a movie with the same title+year already exists and the file has a detected language,
// the file is added as a language variant instead of creating a new movie entry.
func ProcessMovie(
	ctx context.Context,
	deps *Deps,
	libraryID int64,
	result *scanner.ScanResult,
	checkpoint *scanner.ScanCheckpoint,
	existingMediaCache *sync.Map,
) (*int64, error) {
	// Handle nil checkpoint (legacy code path or tests)
	if checkpoint == nil {
		checkpoint = &scanner.ScanCheckpoint{
			FilePath: result.FilePath,
			FileHash: "",
			FileSize: 0,
		}
	}

	// Skip audio files in Movie libraries - they can't be movies (e.g., soundtrack files)
	ext := strings.ToLower(strings.TrimPrefix(result.FilePath[strings.LastIndex(result.FilePath, "."):], "."))
	if scanutil.IsAudioFile(ext) {
		return nil, nil
	}

	// Coordinator already parsed the filename - just use the results
	now := time.Now()
	movie := &media.Movie{
		Media: media.Media{
			LibraryID:       libraryID,
			Title:           result.Title,
			FilePath:        result.FilePath,
			FileSize:        checkpoint.FileSize,
			FileHash:        checkpoint.FileHash,
			Duration:        int(result.Duration),
			IsExtra:         scanutil.IsExtra(result.FilePath),
			Width:           result.Width,
			Height:          result.Height,
			VideoCodec:      result.VideoCodec,
			AudioCodec:      result.AudioCodec,
			Bitrate:         result.Bitrate,
			FrameRate:       result.FrameRate,
			ContainerFormat: result.ContainerFormat,
			Language:        result.Language,
			DateAdded:       now,
			DateModified:    &result.FileMTime,
			CreatedAt:       now,
			UpdatedAt:       now,
		},
	}

	// Set year from scan result
	if result.Year != nil {
		movie.Year = *result.Year
	}

	// NOTE: NFO metadata enrichment is now handled asynchronously by the enrichment pipeline.
	// The scanner saves minimal records; the NFO enricher populates rich metadata.

	// Ensure SortTitle is always set with normalized value
	if movie.SortTitle == "" {
		movie.SortTitle = domainCommon.NormalizeSortTitle(movie.Media.Title)
	}

	// Check for existing movie with the same title+year to either:
	// a) Add as a language variant (different file, has language), or
	// b) Handle file replacement (same file path, or same file re-scanned)
	if _, found := existingMediaCache.Load(result.FilePath); !found {
		existingID, oldPath, err := deps.MediaRepos.Movie.FindByTitleAndYear(ctx, libraryID, result.Title, result.Year)
		if err == nil && existingID > 0 && oldPath != result.FilePath {
			// Found existing movie with same title+year but different file path.
			if result.Language != "" {
				// This file has a detected language — add as a language variant of the existing movie.
				return addLanguageVariant(ctx, deps, libraryID, existingID, result, checkpoint, existingMediaCache)
			}

			// No language detected — treat as file replacement (update existing record).
			deps.Logger.Info("detected file replacement",
				slog.String("title", result.Title),
				slog.String("old_path", oldPath),
				slog.String("new_path", result.FilePath))
			existingMediaCache.Store(result.FilePath, existingID)
		} else if err != nil && !errors.Is(err, media.ErrMediaNotFound) {
			deps.Logger.Warn("failed to check for file replacement",
				slog.String("title", result.Title),
				slog.Any("error", err))
		}
	}

	// Use shared cache-based upsert pattern with race condition handling
	movie.Media.Type = "movie"
	return ProcessMediaWithCache(ctx, deps, libraryID, result.FilePath, existingMediaCache, UpsertCallbacks{
		GetMediaID: func() int64 { return movie.Media.ID },
		SetMediaID: func(id int64) { movie.Media.ID = id },
		Update: func(ctx context.Context) error {
			if err := deps.MediaRepos.Media.Update(ctx, &movie.Media); err != nil {
				return fmt.Errorf("failed to update base media record: %w", err)
			}
			if err := deps.MediaRepos.Movie.UpdateMovie(ctx, movie); err != nil {
				return fmt.Errorf("failed to update movie metadata: %w", err)
			}
			return nil
		},
		Create: func(ctx context.Context) error {
			if err := deps.MediaRepos.Movie.CreateMovie(ctx, movie); err != nil {
				return fmt.Errorf("failed to create movie: %w", err)
			}
			return nil
		},
		PostSave: func(ctx context.Context) {
			PersistMediaTracks(ctx, deps, movie.Media.ID, result)
			priority := pipeline.CalculatePriorityFromMetadata(result.Year, time.Now())
			enqueueForEnrichment(ctx, deps, movie.Media.ID, libraryID, enrichment.MediaTypeMovie, priority)
		},
		EventMeta: &EventMetadata{Type: "movie", Title: movie.Media.Title},
	})
}

// addLanguageVariant creates a new media entry as a variant of an existing movie.
// The variant media entry shares the same movie metadata (title, year, enrichment)
// but has its own file path, technical metadata, tracks, and language tag.
func addLanguageVariant(
	ctx context.Context,
	deps *Deps,
	libraryID int64,
	primaryMediaID int64,
	result *scanner.ScanResult,
	checkpoint *scanner.ScanCheckpoint,
	existingMediaCache *sync.Map,
) (*int64, error) {
	if checkpoint == nil {
		checkpoint = &scanner.ScanCheckpoint{
			FilePath: result.FilePath,
			FileHash: "",
			FileSize: 0,
		}
	}

	now := time.Now()
	variant := &media.Media{
		LibraryID:       libraryID,
		Title:           result.Title,
		FilePath:        result.FilePath,
		FileSize:        checkpoint.FileSize,
		FileHash:        checkpoint.FileHash,
		Duration:        int(result.Duration),
		IsExtra:         scanutil.IsExtra(result.FilePath),
		Width:           result.Width,
		Height:          result.Height,
		VideoCodec:      result.VideoCodec,
		AudioCodec:      result.AudioCodec,
		Bitrate:         result.Bitrate,
		FrameRate:       result.FrameRate,
		ContainerFormat: result.ContainerFormat,
		Type:            "movie",
		Language:        result.Language,
		VariantGroupID:  &primaryMediaID,
		DateAdded:       now,
		DateModified:    &result.FileMTime,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	// Create just the media entry (no movies row — metadata is shared with primary)
	if err := deps.MediaRepos.Media.Create(ctx, variant); err != nil {
		return nil, fmt.Errorf("failed to create variant media record: %w", err)
	}

	// Update the primary media record to set its variant_group_id to its own ID,
	// making it the primary of this variant group. This allows the API to find
	// all variants by querying for media with variant_group_id = primary_id.
	primaryUpdate := &media.Media{
		ID:             primaryMediaID,
		VariantGroupID: &primaryMediaID,
		UpdatedAt:      now,
	}
	if err := deps.MediaRepos.Media.Update(ctx, primaryUpdate); err != nil {
		deps.Logger.Warn("failed to update primary media with variant group ID",
			slog.Int64("primary_media_id", primaryMediaID),
			slog.Any("error", err))
	}

	deps.Logger.Info("added language variant",
		slog.String("title", result.Title),
		slog.String("language", result.Language),
		slog.Int64("primary_media_id", primaryMediaID),
		slog.Int64("variant_media_id", variant.ID))

	// Add to cache for future lookups
	existingMediaCache.Store(result.FilePath, variant.ID)

	// Persist audio and subtitle tracks for this variant
	PersistMediaTracks(ctx, deps, variant.ID, result)

	return &variant.ID, nil
}

// ReconcileVariantGroups scans the library for movies with the same title and year
// that don't have variant_group_id set, and groups them into variant groups.
// This handles cases where variant grouping was missed during initial scan (e.g.,
// first file had no language detected, or files were scanned in different runs).
func ReconcileVariantGroups(ctx context.Context, mediaRepos *scan.MediaRepositories, logger *slog.Logger, libraryID int64) error {
	// Get all movies in the library that don't have variant_group_id set
	movies, err := mediaRepos.Movie.GetMoviesWithoutVariantGroup(ctx, libraryID)
	if err != nil {
		return fmt.Errorf("failed to get movies without variant group: %w", err)
	}

	if len(movies) == 0 {
		return nil
	}

	// Group by normalized title + year
	groups := make(map[string][]*media.Movie)
	for _, m := range movies {
		key := domainCommon.NormalizeSortTitle(m.Media.Title) + "|" + fmt.Sprintf("%d", m.Year)
		groups[key] = append(groups[key], m)
	}

	for key, group := range groups {
		if len(group) < 2 {
			continue // No variants to group
		}

		logger.Info("reconciling variant group",
			slog.String("key", key),
			slog.Int("count", len(group)))

		// Pick the primary: the one with the most audio tracks or first one
		// For simplicity, use the first one as primary
		primary := group[0]
		primaryMediaID := primary.Media.ID

		// Set primary's variant_group_id to its own ID
		primaryUpdate := &media.Media{
			ID:             primaryMediaID,
			VariantGroupID: &primaryMediaID,
			UpdatedAt:      time.Now(),
		}
		if err := mediaRepos.Media.Update(ctx, primaryUpdate); err != nil {
			logger.Warn("failed to update primary media with variant group ID",
				slog.Int64("primary_media_id", primaryMediaID),
				slog.Any("error", err))
			continue
		}

		// Update all other movies in the group to point to the primary
		for _, m := range group[1:] {
			variantUpdate := &media.Media{
				ID:             m.Media.ID,
				VariantGroupID: &primaryMediaID,
				UpdatedAt:      time.Now(),
			}
			if err := mediaRepos.Media.Update(ctx, variantUpdate); err != nil {
				logger.Warn("failed to update variant media with variant group ID",
					slog.Int64("variant_media_id", m.Media.ID),
					slog.Any("error", err))
				continue
			}
		}
	}

	return nil
}
