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

type JobRepo struct {
	coll *mongo.Collection
}

func NewJobRepo(db *mongo.Database) *JobRepo {
	return &JobRepo{coll: db.Collection("jobs")}
}

func (r *JobRepo) Create(ctx context.Context, j *domain.Job) error {
	j.CreatedAt = time.Now()
	j.UpdatedAt = j.CreatedAt
	if j.ID == "" {
		j.ID = primitive.NewObjectID().Hex()
	}
	if j.Status == "" {
		j.Status = domain.JobStatusPending
	}
	_, err := r.coll.InsertOne(ctx, j)
	return err
}

func (r *JobRepo) Get(ctx context.Context, id string) (*domain.Job, error) {
	var j domain.Job
	err := r.coll.FindOne(ctx, bson.M{"_id": id}).Decode(&j)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	return &j, err
}

func (r *JobRepo) Update(ctx context.Context, j *domain.Job) error {
	j.UpdatedAt = time.Now()
	_, err := r.coll.ReplaceOne(ctx, bson.M{"_id": j.ID}, j)
	return err
}

func (r *JobRepo) PickPending(ctx context.Context, jobType string, leaseTime time.Duration) (*domain.Job, error) {
	now := time.Now()
	leaseUntil := now.Add(leaseTime)

	filter := bson.M{
		"type":   jobType,
		"status": domain.JobStatusPending,
		"$or": []bson.M{
			{"leaseUntil": nil},
			{"leaseUntil": bson.M{"$lt": now}},
		},
	}
	update := bson.M{
		"$set": bson.M{
			"status":     domain.JobStatusRunning,
			"leaseUntil": leaseUntil,
			"updatedAt":  now,
		},
		"$inc": bson.M{"attempts": 1},
	}

	var j domain.Job
	err := r.coll.FindOneAndUpdate(ctx, filter, update,
		options.FindOneAndUpdate().SetReturnDocument(options.After),
	).Decode(&j)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	return &j, err
}

func (r *JobRepo) ListPending(ctx context.Context) ([]*domain.Job, error) {
	cursor, err := r.coll.Find(ctx, bson.M{"status": domain.JobStatusPending})
	if err != nil {
		return nil, err
	}
	var jobs []*domain.Job
	if err := cursor.All(ctx, &jobs); err != nil {
		return nil, err
	}
	return jobs, nil
}

func (r *JobRepo) ListStuck(ctx context.Context, threshold time.Duration) ([]*domain.Job, error) {
	cutoff := time.Now().Add(-threshold)
	cursor, err := r.coll.Find(ctx, bson.M{
		"status":    domain.JobStatusRunning,
		"updatedAt": bson.M{"$lt": cutoff},
	})
	if err != nil {
		return nil, err
	}
	var jobs []*domain.Job
	if err := cursor.All(ctx, &jobs); err != nil {
		return nil, err
	}
	return jobs, nil
}
