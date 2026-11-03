package mongo

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/premchand/story-builder/internal/domain"
)

type SceneLockRepo struct {
	coll *mongo.Collection
}

func NewSceneLockRepo(db *mongo.Database) *SceneLockRepo {
	return &SceneLockRepo{coll: db.Collection("scene_locks")}
}

func (r *SceneLockRepo) Acquire(ctx context.Context, lock *domain.SceneLock) (bool, error) {
	lock.AcquiredAt = time.Now()
	opts := options.Update().SetUpsert(true)
	filter := bson.M{
		"_id": lock.SceneID,
		"$or": []bson.M{
			{"ttl": bson.M{"$lt": time.Now()}},
			{"ttl": nil},
		},
	}
	update := bson.M{
		"$set": bson.M{
			"storyId":    lock.StoryID,
			"genId":      lock.GenID,
			"workerId":   lock.WorkerID,
			"acquiredAt": lock.AcquiredAt,
			"ttl":        lock.TTL,
		},
		"$inc": bson.M{"version": 1},
	}
	result, err := r.coll.UpdateOne(ctx, filter, update, opts)
	if err != nil {
		return false, err
	}
	return result.UpsertedCount > 0 || result.ModifiedCount > 0, nil
}

func (r *SceneLockRepo) Release(ctx context.Context, sceneID string) error {
	_, err := r.coll.DeleteOne(ctx, bson.M{"_id": sceneID})
	return err
}

func (r *SceneLockRepo) Get(ctx context.Context, sceneID string) (*domain.SceneLock, error) {
	var lock domain.SceneLock
	err := r.coll.FindOne(ctx, bson.M{"_id": sceneID}).Decode(&lock)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &lock, nil
}
