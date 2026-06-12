package api

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/pgvector/pgvector-go"
	"github.com/premchand/story-builder/internal/canon"
	"github.com/premchand/story-builder/internal/db"
)

type dbLocService struct{ q *db.Queries }

func NewDBLocService(q *db.Queries) *dbLocService {
	return &dbLocService{q: q}
}

func (s *dbLocService) Create(ctx context.Context, name, description string, props []string) (*canon.Location, error) {
	if props == nil {
		props = []string{}
	}
	l, err := s.q.CreateLocation(ctx, db.CreateLocationParams{
		Name:        name,
		Description: description,
		Props:       jsonBytes(props),
	})
	if err != nil {
		return nil, err
	}
	return toDomainLoc(l), nil
}

func (s *dbLocService) Get(ctx context.Context, id uuid.UUID, version int) (*canon.Location, error) {
	if version > 0 {
		l, err := s.q.GetLocationAtVersion(ctx, db.GetLocationAtVersionParams{
			ID:      toUUID(id),
			Version: int32(version),
		})
		if err != nil {
			return nil, err
		}
		return toDomainLoc(l), nil
	}
	l, err := s.q.GetLocationLatest(ctx, toUUID(id))
	if err != nil {
		return nil, err
	}
	return toDomainLocFromLatest(l), nil
}

func (s *dbLocService) Update(ctx context.Context, id uuid.UUID, description string, props []string) (*canon.Location, error) {
	l, err := s.q.UpdateLocation(ctx, db.UpdateLocationParams{
		ID:          toUUID(id),
		Description: description,
		Column3:     jsonBytes(props),
	})
	if err != nil {
		return nil, err
	}
	return toDomainLoc(l), nil
}

func (s *dbLocService) List(ctx context.Context) ([]canon.Location, error) {
	locs, err := s.q.ListLocations(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]canon.Location, len(locs))
	for i, l := range locs {
		result[i] = *toDomainLocFromLatest(l)
	}
	return result, nil
}

func toDomainLoc(l db.Location) *canon.Location {
	var props []string
	json.Unmarshal(l.Props, &props)
	return &canon.Location{
		ID:          fromUUID(l.ID),
		Version:     int(l.Version),
		Name:        l.Name,
		Description: l.Description,
		Props:       props,
		CreatedAt:   l.CreatedAt.Time,
	}
}

func toDomainLocFromLatest(l db.LatestLocation) *canon.Location {
	var props []string
	json.Unmarshal(l.Props, &props)
	return &canon.Location{
		ID:          fromUUID(l.ID),
		Version:     int(l.Version),
		Name:        l.Name,
		Description: l.Description,
		Props:       props,
		CreatedAt:   l.CreatedAt.Time,
	}
}

type dbLoreService struct{ q *db.Queries }

func NewDBLoreService(q *db.Queries) *dbLoreService {
	return &dbLoreService{q: q}
}

func (s *dbLoreService) Create(ctx context.Context, tags []string, content string) (*canon.Lore, error) {
	if tags == nil {
		tags = []string{}
	}
	l, err := s.q.CreateLore(ctx, db.CreateLoreParams{
		Tags:      tags,
		Content:   content,
		Embedding: pgvector.Vector{},
	})
	if err != nil {
		return nil, err
	}
	return &canon.Lore{
		ID:        fromUUID(l.ID),
		Tags:      l.Tags,
		Content:   l.Content,
		CreatedAt: l.CreatedAt.Time,
	}, nil
}

func (s *dbLoreService) List(ctx context.Context) ([]canon.Lore, error) {
	items, err := s.q.ListLore(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]canon.Lore, len(items))
	for i, l := range items {
		result[i] = canon.Lore{
			ID:        fromUUID(l.ID),
			Tags:      l.Tags,
			Content:   l.Content,
			CreatedAt: l.CreatedAt.Time,
		}
	}
	return result, nil
}

func (s *dbLoreService) SearchByTags(ctx context.Context, tags []string) ([]canon.Lore, error) {
	items, err := s.q.SearchLoreByTags(ctx, tags)
	if err != nil {
		return nil, err
	}
	result := make([]canon.Lore, len(items))
	for i, l := range items {
		result[i] = canon.Lore{
			ID:        fromUUID(l.ID),
			Tags:      l.Tags,
			Content:   l.Content,
			CreatedAt: l.CreatedAt.Time,
		}
	}
	return result, nil
}

func (s *dbLoreService) SearchSimilar(ctx context.Context, embedding []float32, limit int) ([]canon.Lore, error) {
	vec := pgvector.NewVector(embedding)
	items, err := s.q.SearchLoreSimilar(ctx, db.SearchLoreSimilarParams{
		Column1: vec,
		Limit:   int32(limit),
	})
	if err != nil {
		return nil, err
	}
	result := make([]canon.Lore, len(items))
	for i, l := range items {
		result[i] = canon.Lore{
			ID:        fromUUID(l.ID),
			Tags:      l.Tags,
			Content:   l.Content,
			CreatedAt: l.CreatedAt.Time,
		}
	}
	return result, nil
}
