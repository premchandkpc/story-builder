package mongo

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/premchand/story-builder/internal/ledger"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type CharacterStateDoc struct {
	ID            string            `bson:"_id" json:"id"`
	StoryID       string            `bson:"story_id" json:"story_id"`
	CharacterID   string            `bson:"character_id" json:"character_id"`
	AsOfScene     string            `bson:"as_of_scene" json:"as_of_scene"`
	Location      string            `bson:"location,omitempty" json:"location,omitempty"`
	Knows         []string          `bson:"knows,omitempty" json:"knows,omitempty"`
	DoesNotKnow   []string          `bson:"does_not_know,omitempty" json:"does_not_know,omitempty"`
	Mood          string            `bson:"mood,omitempty" json:"mood,omitempty"`
	Relationships map[string]string `bson:"relationships,omitempty" json:"relationships,omitempty"`
	Items         []string          `bson:"items,omitempty" json:"items,omitempty"`
	UpdatedAt     time.Time         `bson:"updated_at" json:"updated_at"`
}

type StateRepo struct {
	coll *mongo.Collection
}

func NewStateRepo(client *Client) *StateRepo {
	return &StateRepo{
		coll: client.Coll(CollCharSceneState),
	}
}

func (r *StateRepo) Upsert(ctx context.Context, doc *CharacterStateDoc) error {
	id := doc.StoryID + ":" + doc.CharacterID + ":" + doc.AsOfScene
	doc.ID = id
	doc.UpdatedAt = time.Now()
	_, err := r.coll.ReplaceOne(
		ctx,
		bson.M{
			"story_id":     doc.StoryID,
			"character_id": doc.CharacterID,
			"as_of_scene":  doc.AsOfScene,
		},
		doc,
		options.Replace().SetUpsert(true),
	)
	return err
}

func (r *StateRepo) GetByScene(ctx context.Context, storyID, sceneID string) ([]CharacterStateDoc, error) {
	cursor, err := r.coll.Find(ctx, bson.M{
		"story_id":   storyID,
		"as_of_scene": sceneID,
	})
	if err != nil {
		return nil, err
	}
	var docs []CharacterStateDoc
	if err := cursor.All(ctx, &docs); err != nil {
		return nil, err
	}
	return docs, nil
}

type MongoStateWriter struct {
	repo *StateRepo
}

func NewMongoStateWriter(client *Client) *MongoStateWriter {
	return &MongoStateWriter{repo: NewStateRepo(client)}
}

func (w *MongoStateWriter) UpsertState(ctx context.Context, storyID, characterID, asOfScene uuid.UUID, state ledger.CharacterState) error {
	doc := &CharacterStateDoc{
		StoryID:       storyID.String(),
		CharacterID:   characterID.String(),
		AsOfScene:     asOfScene.String(),
		Location:      state.Location,
		Knows:         state.Knows,
		DoesNotKnow:   state.DoesNotKnow,
		Mood:          state.Mood,
		Relationships: state.Relationships,
		Items:         state.Items,
	}
	return w.repo.Upsert(ctx, doc)
}
