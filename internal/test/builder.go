package test

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"

	"github.com/premchand/story-builder/internal/domain"
)

type StoryBuilder struct {
	story  *domain.Story
	scenes []*domain.Scene
	edges  []*domain.SceneEdge
	chars  []*domain.Character
}

func NewStory(title string) *StoryBuilder {
	return &StoryBuilder{
		story: &domain.Story{
			ID: primitive.NewObjectID().Hex(), Title: title,
			CreatedAt: time.Now(), UpdatedAt: time.Now(),
		},
	}
}

func (b *StoryBuilder) WithScene(id, title string, opts ...SceneOpt) *StoryBuilder {
	scene := &domain.Scene{
		ID: id, StoryID: b.story.ID, Title: title,
		Status: domain.SceneStatusDraft, FlowType: "dialogue",
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	for _, opt := range opts {
		opt(scene)
	}
	b.scenes = append(b.scenes, scene)
	return b
}

func (b *StoryBuilder) WithEdge(from, to, edgeType string) *StoryBuilder {
	b.edges = append(b.edges, &domain.SceneEdge{
		ID: primitive.NewObjectID().Hex(), StoryID: b.story.ID,
		FromSceneID: from, ToSceneID: to, Type: edgeType,
	})
	return b
}

func (b *StoryBuilder) WithCharacter(charID, name string) *StoryBuilder {
	b.chars = append(b.chars, &domain.Character{
		CharID: charID, StoryID: b.story.ID, Name: name,
		CreatedAt: time.Now(),
	})
	return b
}

func (b *StoryBuilder) Story() *domain.Story     { return b.story }
func (b *StoryBuilder) Scenes() []*domain.Scene   { return b.scenes }
func (b *StoryBuilder) Edges() []*domain.SceneEdge { return b.edges }
func (b *StoryBuilder) Characters() []*domain.Character { return b.chars }

func (b *StoryBuilder) Insert(ctx context.Context, db *mongo.Database) error {
	collections := map[string]any{
		"stories":    b.story,
		"scenes":     b.scenes,
		"scene_edges": b.edges,
		"characters": b.chars,
	}
	for coll, docs := range collections {
		switch d := docs.(type) {
		case *domain.Story:
			if _, err := db.Collection(coll).InsertOne(ctx, d); err != nil {
				return err
			}
		case []*domain.Scene:
			for _, doc := range d {
				if _, err := db.Collection(coll).InsertOne(ctx, doc); err != nil {
					return err
				}
			}
		case []*domain.SceneEdge:
			for _, doc := range d {
				if _, err := db.Collection(coll).InsertOne(ctx, doc); err != nil {
					return err
				}
			}
		case []*domain.Character:
			for _, doc := range d {
				if _, err := db.Collection(coll).InsertOne(ctx, doc); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

type SceneOpt func(*domain.Scene)

func WithBeatIntent(s string) SceneOpt    { return func(sc *domain.Scene) { sc.BeatIntent = s } }
func WithPOV(s string) SceneOpt           { return func(sc *domain.Scene) { sc.POV = s } }
func WithTone(s string) SceneOpt          { return func(sc *domain.Scene) { sc.Tone = s } }
func WithFlowType(s string) SceneOpt      { return func(sc *domain.Scene) { sc.FlowType = s } }
func WithStatus(s string) SceneOpt        { return func(sc *domain.Scene) { sc.Status = s } }
func WithParticipants(ids ...string) SceneOpt {
	return func(sc *domain.Scene) { sc.Participants = ids }
}
func WithTimelinePosition(n int) SceneOpt {
	return func(sc *domain.Scene) { sc.TimelinePosition = n }
}
func WithLocation(name string) SceneOpt {
	return func(sc *domain.Scene) { sc.LocationRef = name }
}
