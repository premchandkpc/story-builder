package retrieval

import (
	"context"
	"math"
	"sort"
	"time"

	"github.com/premchand/story-builder/internal/domain"
	"github.com/premchand/story-builder/internal/llm"
	"github.com/premchand/story-builder/internal/repository"
)

type RetrievalConfig struct {
	MaxHotTokens       int
	MaxColdTokens      int
	RelevanceWeight    float64
	ImportanceWeight   float64
	RecencyWeight      float64
	MinFinalScore      float64
	MaxMemoriesPerChar int
}

func DefaultRetrievalConfig() RetrievalConfig {
	return RetrievalConfig{
		MaxHotTokens:       4000,
		MaxColdTokens:      2000,
		RelevanceWeight:    0.5,
		ImportanceWeight:   0.3,
		RecencyWeight:      0.2,
		MinFinalScore:      0.1,
		MaxMemoriesPerChar: 5,
	}
}

type HardContext struct {
	ParticipantIDs []string
	RecentTimeline []domain.TimelineEvent
	SceneGoal      string
}

type ScoredMemory struct {
	Memory     domain.CharacterMemory
	Relevance  float64
	Importance float64
	Recency    float64
	FinalScore float64
}

type HotContext struct {
	Memories   []ScoredMemory
	TokenCount int
}

type ColdSnippet struct {
	Content    string
	Source     string
	TokenCount int
}

type ColdContext struct {
	Snippets   []ColdSnippet
	TokenCount int
}

type RetrievalResult struct {
	HardConstraints HardContext
	HotContext      HotContext
	ColdContext     ColdContext
}

type RetrievalService struct {
	MemoryRepo   repository.MemoryRepository
	TimelineRepo repository.TimelineRepository
	BibleRepo    repository.BibleRepository
	Embedding    llm.EmbeddingService
	Config       RetrievalConfig
}

func NewRetrievalService(
	memRepo repository.MemoryRepository,
	tlRepo repository.TimelineRepository,
	bibleRepo repository.BibleRepository,
	embedding llm.EmbeddingService,
	cfg RetrievalConfig,
) *RetrievalService {
	return &RetrievalService{
		MemoryRepo:   memRepo,
		TimelineRepo: tlRepo,
		BibleRepo:    bibleRepo,
		Embedding:    embedding,
		Config:       cfg,
	}
}

func (r *RetrievalService) RetrieveForScene(ctx context.Context, storyID, sceneID string, charIDs []string, sceneGoal string) (*RetrievalResult, error) {
	result := &RetrievalResult{}

	hard, err := r.fetchHardConstraints(ctx, storyID, sceneID, charIDs, sceneGoal)
	if err != nil {
		return nil, err
	}
	result.HardConstraints = *hard

	hot, err := r.fetchHotContext(ctx, storyID, charIDs, sceneGoal)
	if err != nil {
		return nil, err
	}
	result.HotContext = *hot

	cold, err := r.fetchColdContext(ctx, storyID)
	if err != nil {
		return nil, err
	}
	result.ColdContext = *cold

	return result, nil
}

func (r *RetrievalService) fetchHardConstraints(ctx context.Context, storyID, sceneID string, charIDs []string, sceneGoal string) (*HardContext, error) {
	tlEvents, _ := r.TimelineRepo.ListByStory(ctx, storyID)
	recent := make([]domain.TimelineEvent, 0, len(tlEvents))
	for _, e := range tlEvents {
		if e != nil {
			recent = append(recent, *e)
		}
	}
	if len(recent) > 3 {
		recent = recent[len(recent)-3:]
	}

	return &HardContext{
		ParticipantIDs: charIDs,
		RecentTimeline: recent,
		SceneGoal:      sceneGoal,
	}, nil
}

func (r *RetrievalService) fetchHotContext(ctx context.Context, storyID string, charIDs []string, query string) (*HotContext, error) {
	var allMemories []ScoredMemory
	now := time.Now()

	for _, charID := range charIDs {
		memories, _ := r.MemoryRepo.ListByCharacter(ctx, charID)
		for _, mem := range memories {
			if mem == nil {
				continue
			}
			score := r.scoreMemory(mem, now)
			if score.FinalScore >= r.Config.MinFinalScore {
				allMemories = append(allMemories, *score)
			}
		}
	}

	sort.Slice(allMemories, func(i, j int) bool {
		return allMemories[i].FinalScore > allMemories[j].FinalScore
	})

	maxMemories := r.Config.MaxMemoriesPerChar * len(charIDs)
	if len(allMemories) > maxMemories {
		allMemories = allMemories[:maxMemories]
	}

	return &HotContext{
		Memories:   allMemories,
		TokenCount: len(allMemories) * 50,
	}, nil
}

func (r *RetrievalService) fetchColdContext(ctx context.Context, storyID string) (*ColdContext, error) {
	bible, err := r.BibleRepo.GetByStory(ctx, storyID)
	if err != nil || bible == nil {
		return &ColdContext{}, nil
	}

	var snippets []ColdSnippet
	if bible.World != "" {
		snippets = append(snippets, ColdSnippet{
			Content:    bible.World,
			Source:     "world",
			TokenCount: len(bible.World) / 4,
		})
	}
	if len(bible.Factions) > 0 {
		for _, f := range bible.Factions {
			snippets = append(snippets, ColdSnippet{
				Content:    f.Name + ": " + f.Goal,
				Source:     "faction",
				TokenCount: (len(f.Name) + len(f.Goal)) / 4,
			})
		}
	}
	if len(bible.MagicSystems) > 0 {
		for _, m := range bible.MagicSystems {
			snippets = append(snippets, ColdSnippet{
				Content:    m.Name + ": " + m.Source,
				Source:     "magic",
				TokenCount: (len(m.Name) + len(m.Source)) / 4,
			})
		}
	}

	var tokenTotal int
	var filtered []ColdSnippet
	for _, s := range snippets {
		if tokenTotal+s.TokenCount <= r.Config.MaxColdTokens || tokenTotal == 0 {
			filtered = append(filtered, s)
			tokenTotal += s.TokenCount
		}
	}

	return &ColdContext{
		Snippets:   filtered,
		TokenCount: tokenTotal,
	}, nil
}

func (r *RetrievalService) scoreMemory(mem *domain.CharacterMemory, now time.Time) *ScoredMemory {
	importance := mem.Importance
	if importance < 0 {
		importance = 0
	}
	if importance > 1 {
		importance = 1
	}

	age := now.Sub(mem.CreatedAt)
	maxAge := 30 * 24 * time.Hour
	recency := 1.0 - float64(age)/float64(maxAge)
	if recency < 0 {
		recency = 0
	}

	relevance := importance

	finalScore := relevance*r.Config.RelevanceWeight +
		importance*r.Config.ImportanceWeight +
		recency*r.Config.RecencyWeight

	return &ScoredMemory{
		Memory:     *mem,
		Relevance:  math.Round(relevance*100) / 100,
		Importance: math.Round(importance*100) / 100,
		Recency:    math.Round(recency*100) / 100,
		FinalScore: math.Round(finalScore*100) / 100,
	}
}
