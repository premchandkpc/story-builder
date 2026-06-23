package mongo

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/premchand/story-builder/internal/domain"
)

type RunRepo struct {
	coll *mongo.Collection
}

func NewRunRepo(db *mongo.Database) *RunRepo {
	return &RunRepo{coll: db.Collection("story_runs")}
}

func (r *RunRepo) Create(ctx context.Context, run *domain.StoryRun) error {
	run.CreatedAt = time.Now()
	run.UpdatedAt = run.CreatedAt
	if run.ID == "" {
		run.ID = primitive.NewObjectID().Hex()
	}
	if run.Status == "" {
		run.Status = domain.RunStatusQueued
	}
	_, err := r.coll.InsertOne(ctx, run)
	return err
}

func (r *RunRepo) Get(ctx context.Context, id string) (*domain.StoryRun, error) {
	var run domain.StoryRun
	err := r.coll.FindOne(ctx, bson.M{"_id": id}).Decode(&run)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &run, nil
}

func (r *RunRepo) ListByStory(ctx context.Context, storyID string, limit int) ([]*domain.StoryRun, error) {
	opt := options.Find().SetSort(bson.D{{Key: "createdAt", Value: -1}})
	if limit > 0 {
		opt.SetLimit(int64(limit))
	}
	cursor, err := r.coll.Find(ctx, bson.M{"storyId": storyID}, opt)
	if err != nil {
		return nil, err
	}
	var runs []*domain.StoryRun
	if err := cursor.All(ctx, &runs); err != nil {
		return nil, err
	}
	return runs, nil
}

func (r *RunRepo) ListByScene(ctx context.Context, sceneID string, limit int) ([]*domain.StoryRun, error) {
	opt := options.Find().SetSort(bson.D{{Key: "createdAt", Value: -1}})
	if limit > 0 {
		opt.SetLimit(int64(limit))
	}
	cursor, err := r.coll.Find(ctx, bson.M{"sceneId": sceneID}, opt)
	if err != nil {
		return nil, err
	}
	var runs []*domain.StoryRun
	if err := cursor.All(ctx, &runs); err != nil {
		return nil, err
	}
	return runs, nil
}

func (r *RunRepo) Update(ctx context.Context, run *domain.StoryRun) error {
	run.UpdatedAt = time.Now()
	_, err := r.coll.ReplaceOne(ctx, bson.M{"_id": run.ID}, run)
	return err
}

func (r *RunRepo) DeleteByStory(ctx context.Context, storyID string) error {
	_, err := r.coll.DeleteMany(ctx, bson.M{"storyId": storyID})
	return err
}

type RunStepRepo struct {
	coll *mongo.Collection
}

func NewRunStepRepo(db *mongo.Database) *RunStepRepo {
	return &RunStepRepo{coll: db.Collection("run_steps")}
}

func (r *RunStepRepo) Create(ctx context.Context, s *domain.RunStep) error {
	s.CreatedAt = time.Now()
	if s.ID == "" {
		s.ID = primitive.NewObjectID().Hex()
	}
	if s.Status == "" {
		s.Status = domain.StepStatusPending
	}
	_, err := r.coll.InsertOne(ctx, s)
	return err
}

func (r *RunStepRepo) ListByRun(ctx context.Context, runID string) ([]*domain.RunStep, error) {
	cursor, err := r.coll.Find(ctx, bson.M{"runId": runID}, options.Find().SetSort(bson.D{{Key: "createdAt", Value: 1}}))
	if err != nil {
		return nil, err
	}
	var steps []*domain.RunStep
	if err := cursor.All(ctx, &steps); err != nil {
		return nil, err
	}
	return steps, nil
}

func (r *RunStepRepo) DeleteByRun(ctx context.Context, runID string) error {
	_, err := r.coll.DeleteMany(ctx, bson.M{"runId": runID})
	return err
}
