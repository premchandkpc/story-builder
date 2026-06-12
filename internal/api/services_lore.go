package api

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/premchand/story-builder/internal/canon"
)

type locService struct {
	locs    []canon.Location
	version map[uuid.UUID]int
}

func NewLocService() *locService {
	return &locService{version: make(map[uuid.UUID]int)}
}

func (s *locService) Create(ctx context.Context, name, description string, props []string) (*canon.Location, error) {
	l := canon.Location{
		ID:          uuid.New(),
		Version:     1,
		Name:        name,
		Description: description,
		Props:       props,
		CreatedAt:   time.Now(),
	}
	s.locs = append(s.locs, l)
	s.version[l.ID] = 2
	return &l, nil
}

func (s *locService) Get(ctx context.Context, id uuid.UUID, version int) (*canon.Location, error) {
	var latest *canon.Location
	for i := range s.locs {
		if s.locs[i].ID == id {
			if version > 0 && s.locs[i].Version == version {
				return &s.locs[i], nil
			}
			if latest == nil || s.locs[i].Version > latest.Version {
				latest = &s.locs[i]
			}
		}
	}
	if latest == nil {
		return nil, fmt.Errorf("location %s not found", id)
	}
	return latest, nil
}

func (s *locService) Update(ctx context.Context, id uuid.UUID, description string, props []string) (*canon.Location, error) {
	next := s.version[id]
	if next == 0 {
		return nil, fmt.Errorf("location %s not found", id)
	}
	var curName string
	for i := len(s.locs) - 1; i >= 0; i-- {
		if s.locs[i].ID == id {
			curName = s.locs[i].Name
			break
		}
	}
	l := canon.Location{
		ID:          id,
		Version:     next,
		Name:        curName,
		Description: description,
		Props:       props,
		CreatedAt:   time.Now(),
	}
	s.locs = append(s.locs, l)
	s.version[id] = next + 1
	return &l, nil
}

func (s *locService) List(ctx context.Context) ([]canon.Location, error) {
	latest := make(map[uuid.UUID]canon.Location)
	for _, l := range s.locs {
		if existing, ok := latest[l.ID]; !ok || l.Version > existing.Version {
			latest[l.ID] = l
		}
	}
	result := make([]canon.Location, 0, len(latest))
	for _, l := range latest {
		result = append(result, l)
	}
	return result, nil
}

type loreService struct {
	items []canon.Lore
}

func NewLoreService() *loreService {
	return &loreService{}
}

func (s *loreService) Create(ctx context.Context, tags []string, content string) (*canon.Lore, error) {
	l := canon.Lore{
		ID:        uuid.New(),
		Tags:      tags,
		Content:   content,
		CreatedAt: time.Now(),
	}
	s.items = append(s.items, l)
	return &l, nil
}

func (s *loreService) List(ctx context.Context) ([]canon.Lore, error) {
	r := make([]canon.Lore, len(s.items))
	copy(r, s.items)
	return r, nil
}

func (s *loreService) SearchByTags(ctx context.Context, tags []string) ([]canon.Lore, error) {
	tagSet := make(map[string]bool, len(tags))
	for _, t := range tags {
		tagSet[t] = true
	}
	var result []canon.Lore
	for _, l := range s.items {
		for _, t := range l.Tags {
			if tagSet[t] {
				result = append(result, l)
				break
			}
		}
	}
	return result, nil
}

func (s *loreService) SearchSimilar(ctx context.Context, embedding []float32, limit int) ([]canon.Lore, error) {
	if limit > len(s.items) {
		limit = len(s.items)
	}
	return s.items[:limit], nil
}
