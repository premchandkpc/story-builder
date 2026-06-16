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

type MemoryRepo struct {
	coll *mongo.Collection
}

func NewMemoryRepo(db *mongo.Database) *MemoryRepo {
	return &MemoryRepo{coll: db.Collection("character_memories")}
}

func (r *MemoryRepo) Create(ctx context.Context, m *domain.CharacterMemory) error {
	m.CreatedAt = time.Now()
	if m.ID == "" {
		m.ID = primitive.NewObjectID().Hex()
	}
	_, err := r.coll.InsertOne(ctx, m)
	return err
}

func (r *MemoryRepo) ListByCharacter(ctx context.Context, characterID string) ([]*domain.CharacterMemory, error) {
	cursor, err := r.coll.Find(ctx, bson.M{"characterId": characterID}, options.Find().SetSort(bson.D{{Key: "createdAt", Value: -1}}))
	if err != nil {
		return nil, err
	}
	var mems []*domain.CharacterMemory
	if err := cursor.All(ctx, &mems); err != nil {
		return nil, err
	}
	return mems, nil
}

func (r *MemoryRepo) Search(ctx context.Context, storyID, characterID string, query []float64, limit int) ([]*domain.CharacterMemory, error) {
	filter := bson.M{"storyId": storyID, "characterId": characterID}

	// When query vector is provided, use MongoDB Atlas Search aggregation
	if len(query) > 0 {
		pipeline := mongo.Pipeline{
			{{Key: "$match", Value: filter}},
			{{Key: "$vectorSearch", Value: bson.M{
				"queryVector":  query,
				"path":         "embedding",
				"numCandidates": limit * 10,
				"limit":        limit,
				"index":        "memory_vector_index",
			}}},
		}
		cursor, err := r.coll.Aggregate(ctx, pipeline)
		if err != nil {
			return r.searchByImportance(ctx, storyID, characterID, limit)
		}
		var mems []*domain.CharacterMemory
		if err := cursor.All(ctx, &mems); err != nil {
			return nil, err
		}
		return mems, nil
	}

	return r.searchByImportance(ctx, storyID, characterID, limit)
}

func (r *MemoryRepo) searchByImportance(ctx context.Context, storyID, characterID string, limit int) ([]*domain.CharacterMemory, error) {
	cursor, err := r.coll.Find(ctx, bson.M{"storyId": storyID, "characterId": characterID},
		options.Find().SetSort(bson.D{{Key: "importance", Value: -1}}).SetLimit(int64(limit)))
	if err != nil {
		return nil, err
	}
	var mems []*domain.CharacterMemory
	if err := cursor.All(ctx, &mems); err != nil {
		return nil, err
	}
	return mems, nil
}
