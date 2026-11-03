package service

import (
	"context"
	"fmt"

	"github.com/premchand/story-builder/internal/domain"
	"github.com/premchand/story-builder/internal/repository"
)

type DiffService struct {
	GenRepo   repository.GenerationRepository
	EventRepo repository.NarrativeEventRepository
}

type GenDiff struct {
	GenAID     string       `json:"genAId"`
	GenBID     string       `json:"genBId"`
	ProseDiff  string       `json:"proseDiff"`
	EventDiffs []EventDiff  `json:"eventDiffs"`
	TokenDiff  struct {
		A int `json:"a"`
		B int `json:"b"`
	} `json:"tokenDiff"`
}

type EventDiff struct {
	EventType string                     `json:"eventType"`
	A         *NarrativeEventSnapshot    `json:"a,omitempty"`
	B         *NarrativeEventSnapshot    `json:"b,omitempty"`
}

type NarrativeEventSnapshot struct {
	SubjectType string         `json:"subjectType"`
	SubjectID   string         `json:"subjectId"`
	Payload     map[string]any `json:"payload"`
	Confidence  float64        `json:"confidence"`
}

func NewDiffService(genRepo repository.GenerationRepository, eventRepo repository.NarrativeEventRepository) *DiffService {
	return &DiffService{GenRepo: genRepo, EventRepo: eventRepo}
}

func (s *DiffService) GenDiff(ctx context.Context, storyID, sceneID, genAID, genBID string) (*GenDiff, error) {
	genA, err := s.GenRepo.Get(ctx, genAID)
	if err != nil || genA == nil {
		return nil, fmt.Errorf("generation A not found: %w", err)
	}
	genB, err := s.GenRepo.Get(ctx, genBID)
	if err != nil || genB == nil {
		return nil, fmt.Errorf("generation B not found: %w", err)
	}

	eventsA, _ := s.EventRepo.ListByScene(ctx, sceneID, 1000)
	eventsB, _ := s.EventRepo.ListByScene(ctx, sceneID, 1000)

	diff := &GenDiff{
		GenAID:     genAID,
		GenBID:     genBID,
		ProseDiff:  buildProseDiff(genA.Output, genB.Output),
		EventDiffs: buildEventDiffs(eventsA, eventsB),
	}
	diff.TokenDiff.A = genA.TotalTokens
	diff.TokenDiff.B = genB.TotalTokens

	return diff, nil
}

func buildProseDiff(a, b string) string {
	if a == b {
		return ""
	}
	maxLen := len(a)
	if len(b) > maxLen {
		maxLen = len(b)
	}
	if maxLen > 500 {
		maxLen = 500
	}

	diffLen := 0
	for i := 0; i < maxLen && i < len(a) && i < len(b); i++ {
		if a[i] != b[i] {
			break
		}
		diffLen = i + 1
	}

	aEnd := a
	if len(aEnd) > diffLen+200 {
		aEnd = aEnd[:diffLen+200] + "..."
	}
	bEnd := b
	if len(bEnd) > diffLen+200 {
		bEnd = bEnd[:diffLen+200] + "..."
	}

	return fmt.Sprintf("--- a\n+++ b\n@@ -%d +%d @@\n %s\n+%s", diffLen, diffLen, aEnd, bEnd)
}

func buildEventDiffs(eventsA, eventsB []*domain.NarrativeEvent) []EventDiff {
	indexA := make(map[string]*domain.NarrativeEvent)
	for _, e := range eventsA {
		indexA[e.ID] = e
	}
	indexB := make(map[string]*domain.NarrativeEvent)
	for _, e := range eventsB {
		indexB[e.ID] = e
	}

	allIDs := make(map[string]bool)
	for _, e := range eventsA {
		allIDs[e.ID] = true
	}
	for _, e := range eventsB {
		allIDs[e.ID] = true
	}

	var diffs []EventDiff
	for id := range allIDs {
		eA, inA := indexA[id]
		eB, inB := indexB[id]

		if inA && !inB {
			diffs = append(diffs, EventDiff{
				EventType: "removed",
				A:         eventToSnapshot(eA),
			})
		} else if !inA && inB {
			diffs = append(diffs, EventDiff{
				EventType: "added",
				B:         eventToSnapshot(eB),
			})
		}
	}

	return diffs
}

func eventToSnapshot(e *domain.NarrativeEvent) *NarrativeEventSnapshot {
	if e == nil {
		return nil
	}
	return &NarrativeEventSnapshot{
		SubjectType: e.SubjectType,
		SubjectID:   e.SubjectID,
		Payload:     e.Payload,
		Confidence:  e.Confidence,
	}
}
