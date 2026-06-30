package movies

import (
	"context"
	"fmt"

	"github.com/mantonx/viewra/internal/domain/media"
)

// GetMovieUseCase handles the business logic for getting a single movie
type GetMovieUseCase struct {
	repo      media.MovieRepository
	mediaRepo media.Repository
}

// NewGetMovieUseCase creates a new instance of GetMovieUseCase
func NewGetMovieUseCase(repo media.MovieRepository, mediaRepo media.Repository) *GetMovieUseCase {
	return &GetMovieUseCase{
		repo:      repo,
		mediaRepo: mediaRepo,
	}
}

// Execute retrieves a movie by its ID
func (uc *GetMovieUseCase) Execute(ctx context.Context, id int64) (*MovieResponse, error) {
	movie, err := uc.repo.GetMovieByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get movie: %w", err)
	}

	response := ToMovieResponse(movie)

	// Fetch language variants if this media has a variant group
	if movie.VariantGroupID != nil {
		variants, err := uc.mediaRepo.GetVariantsByGroupID(ctx, *movie.VariantGroupID)
		if err != nil {
			return nil, fmt.Errorf("failed to get variants: %w", err)
		}
		response.Variants = make([]MediaVariantResponse, 0, len(variants))
		for _, v := range variants {
			if v.ID == id {
				continue // skip self
			}
			response.Variants = append(response.Variants, MediaVariantResponse{
				ID:             v.ID,
				Language:       v.Language,
				FilePath:       v.FilePath,
				FileSize:       v.FileSize,
				Duration:       v.Duration,
				Width:          v.Width,
				Height:         v.Height,
				VideoCodec:     v.VideoCodec,
				AudioCodec:     v.AudioCodec,
				ContainerFormat: v.ContainerFormat,
			})
		}
	} else {
		// This movie is the primary; check if it has variants pointing to it
		variants, err := uc.mediaRepo.GetVariantsByGroupID(ctx, id)
		if err != nil {
			return nil, fmt.Errorf("failed to get variants: %w", err)
		}
		response.Variants = make([]MediaVariantResponse, 0, len(variants))
		for _, v := range variants {
			response.Variants = append(response.Variants, MediaVariantResponse{
				ID:             v.ID,
				Language:       v.Language,
				FilePath:       v.FilePath,
				FileSize:       v.FileSize,
				Duration:       v.Duration,
				Width:          v.Width,
				Height:         v.Height,
				VideoCodec:     v.VideoCodec,
				AudioCodec:     v.AudioCodec,
				ContainerFormat: v.ContainerFormat,
			})
		}
	}

	return &response, nil
}
