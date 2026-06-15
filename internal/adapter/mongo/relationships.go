package mongo

import (
	"context"
	"time"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type RelationshipDoc struct {
	ID         string              `bson:"_id" json:"id"`
	StoryID    string              `bson:"story_id" json:"story_id"`
	CharA      string              `bson:"char_a" json:"char_a"`
	CharB      string              `bson:"char_b" json:"char_b"`
	Trust      float64             `bson:"trust" json:"trust"`
	Respect    float64             `bson:"respect" json:"respect"`
	Fear       float64             `bson:"fear" json:"fear"`
	Affection  float64             `bson:"affection" json:"affection"`
	Dependency float64             `bson:"dependency,omitempty" json:"dependency,omitempty"`
	Rivalry    float64             `bson:"rivalry,omitempty" json:"rivalry,omitempty"`
	Loyalty    float64             `bson:"loyalty,omitempty" json:"loyalty,omitempty"`
	Suspicion  float64             `bson:"suspicion,omitempty" json:"suspicion,omitempty"`
	History    []map[string]any   `bson:"history,omitempty" json:"history,omitempty"`
	UpdatedAt  time.Time           `bson:"updated_at" json:"updated_at"`
}

type RelationshipRepo struct {
	coll       *mongo.Collection
	histColl   *mongo.Collection
}

func NewRelationshipRepo(client *Client) *RelationshipRepo {
	return &RelationshipRepo{
		coll:     client.Coll(CollRelationships),
		histColl: client.Coll(CollRelHistory),
	}
}

func (r *RelationshipRepo) Upsert(ctx context.Context, doc *RelationshipDoc) error {
	doc.ID = uuid.New().String()
	doc.UpdatedAt = time.Now()
	filter := bson.M{"story_id": doc.StoryID, "char_a": doc.CharA, "char_b": doc.CharB}
	_, err := r.coll.ReplaceOne(ctx, filter, doc, options.Replace().SetUpsert(true))
	return err
}

func (r *RelationshipRepo) Get(ctx context.Context, storyID, charA, charB string) (*RelationshipDoc, error) {
	var doc RelationshipDoc
	err := r.coll.FindOne(ctx, bson.M{
		"story_id": storyID,
		"$or": []bson.M{
			{"char_a": charA, "char_b": charB},
			{"char_a": charB, "char_b": charA},
		},
	}).Decode(&doc)
	if err != nil {
		return nil, err
	}
	return &doc, nil
}

func (r *RelationshipRepo) ListForCharacter(ctx context.Context, storyID, charID string) ([]RelationshipDoc, error) {
	cursor, err := r.coll.Find(ctx, bson.M{
		"story_id": storyID,
		"$or": []bson.M{
			{"char_a": charID},
			{"char_b": charID},
		},
	})
	if err != nil {
		return nil, err
	}
	var docs []RelationshipDoc
	if err := cursor.All(ctx, &docs); err != nil {
		return nil, err
	}
	return docs, nil
}

func (r *RelationshipRepo) AppendHistory(ctx context.Context, storyID, charA, charB string, entry map[string]any) error {
	_, err := r.histColl.InsertOne(ctx, entry)
	return err
}

func (r *RelationshipRepo) GetHistory(ctx context.Context, storyID, charA, charB string) ([]map[string]any, error) {
	cursor, err := r.histColl.Find(ctx, bson.M{
		"story_id": storyID,
		"$or": []bson.M{
			{"char_a": charA, "char_b": charB},
			{"char_a": charB, "char_b": charA},
		},
	})
	if err != nil {
		return nil, err
	}
	var docs []map[string]any
	if err := cursor.All(ctx, &docs); err != nil {
		return nil, err
	}
	return docs, nil
}
