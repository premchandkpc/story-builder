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

func (s *dbLocService) Create(name, description string, props []string) (*canon.Location, error) {
	if props == nil {
		props = []string{}
	}
	l, err := s.q.CreateLocation(context.Background(), db.CreateLocationParams{
		Name:        name,
		Description: description,
		Props:       jsonBytes(props),
	})
	if err != nil {
		return nil, err
	}
	return toDomainLoc(l), nil
}

func (s *dbLocService) Get(id uuid.UUID, version int) (*canon.Location, error) {
	if version > 0 {
		l, err := s.q.GetLocationAtVersion(context.Background(), db.GetLocationAtVersionParams{
			ID:      toUUID(id),
			Version: int32(version),
		})
		if err != nil {
			return nil, err
		}
		return toDomainLoc(l), nil
	}
	l, err := s.q.GetLocationLatest(context.Background(), toUUID(id))
	if err != nil {
		return nil, err
	}
	return toDomainLocFromLatest(l), nil
}

func (s *dbLocService) Update(id uuid.UUID, description string, props []string) (*canon.Location, error) {
	l, err := s.q.UpdateLocation(context.Background(), db.UpdateLocationParams{
		ID:          toUUID(id),
		Description: description,
		Column3:     jsonBytes(props),
	})
	if err != nil {
		return nil, err
	}
	return toDomainLoc(l), nil
}

func (s *dbLocService) List() ([]canon.Location, error) {
	locs, err := s.q.ListLocations(context.Background())
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

func (s *dbLoreService) Create(tags []string, content string) (*canon.Lore, error) {
	if tags == nil {
		tags = []string{}
	}
	l, err := s.q.CreateLore(context.Background(), db.CreateLoreParams{
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

func (s *dbLoreService) List() ([]canon.Lore, error) {
	items, err := s.q.ListLore(context.Background())
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

func (s *dbLoreService) SearchByTags(tags []string) ([]canon.Lore, error) {
	items, err := s.q.SearchLoreByTags(context.Background(), tags)
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

func (s *dbLoreService) SearchSimilar(embedding []float32, limit int) ([]canon.Lore, error) {
	vec := pgvector.NewVector(embedding)
	items, err := s.q.SearchLoreSimilar(context.Background(), db.SearchLoreSimilarParams{
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
