package builtin

import (
	"context"
	"fmt"

	pluginv1 "github.com/mantonx/viewra/api/proto/plugin"
	appenrich "github.com/mantonx/viewra/internal/application/enrichment"
	"github.com/mantonx/viewra/internal/domain/enrichment"
)

type MetadataEnricher struct {
	enrichers []appenrich.Enricher
}

func NewMetadataEnricher(enrichers ...appenrich.Enricher) *MetadataEnricher {
	return &MetadataEnricher{enrichers: enrichers}
}

func (e *MetadataEnricher) Stage() string { return "metadata" }

func (e *MetadataEnricher) Capabilities() appenrich.EnricherCapabilities {
	types := []enrichment.MediaType{
		enrichment.MediaTypeMovie,
		enrichment.MediaTypeTV,
		enrichment.MediaTypeTVShow,
	}
	return appenrich.NewCapabilitiesBuilder().
		WithMediaTypes(types...).
		WithProvides("metadata", "external_ids", "artwork").
		AsLocal().
		Build()
}

func (e *MetadataEnricher) Enrich(ctx context.Context, req *pluginv1.EnrichRequest) (*pluginv1.EnrichResponse, error) {
	var lastResp *pluginv1.EnrichResponse
	for _, en := range e.enrichers {
		resp, err := en.Enrich(ctx, req)
		if err != nil {
			return nil, fmt.Errorf("metadata: %s: %w", en.Stage(), err)
		}
		if !resp.Skipped {
			lastResp = resp
		}
	}
	return lastResp, nil
}