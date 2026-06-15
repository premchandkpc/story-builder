package mongo

import (
	"context"
	"time"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type CharacterDoc struct {
	ID              string            `bson:"_id" json:"id"`
	Name            string            `bson:"name" json:"name"`
	Description     string            `bson:"description,omitempty" json:"description,omitempty"`
	Archetype       string            `bson:"archetype,omitempty" json:"archetype,omitempty"`
	Personality     map[string]any    `bson:"personality,omitempty" json:"personality,omitempty"`
	Traits          []string          `bson:"traits" json:"traits"`
	VoiceSamples    []string          `bson:"voice_samples" json:"voice_samples"`
	Goals           []map[string]any  `bson:"goals,omitempty" json:"goals,omitempty"`
	Fears           []string          `bson:"fears,omitempty" json:"fears,omitempty"`
	Beliefs         []map[string]any  `bson:"beliefs,omitempty" json:"beliefs,omitempty"`
	VoiceProfile    map[string]any    `bson:"voice_profile,omitempty" json:"voice_profile,omitempty"`
	Appearance      map[string]any    `bson:"appearance,omitempty" json:"appearance,omitempty"`
	Metadata        map[string]any    `bson:"metadata,omitempty" json:"metadata,omitempty"`
	ParentCharID    string            `bson:"parent_character,omitempty" json:"parent_character,omitempty"`
	CreatedAt       time.Time         `bson:"created_at" json:"created_at"`
	UpdatedAt       time.Time         `bson:"updated_at" json:"updated_at"`
}

type CharRuntimeDoc struct {
	ID            string         `bson:"_id" json:"id"`
	CharacterID   string         `bson:"character_id" json:"character_id"`
	StoryID       string         `bson:"story_id" json:"story_id"`
	Emotion       map[string]any `bson:"emotion" json:"emotion"`
	Stress        float64        `bson:"stress" json:"stress"`
	Energy        float64        `bson:"energy" json:"energy"`
	CurrentGoal   string         `bson:"current_goal,omitempty" json:"current_goal,omitempty"`
	ActiveMemIDs  []string       `bson:"active_memories" json:"active_memories"`
	Location      string         `bson:"location,omitempty" json:"location,omitempty"`
	UpdatedAt     time.Time      `bson:"updated_at" json:"updated_at"`
}

type CharSceneStateDoc struct {
	ID          string         `bson:"_id" json:"id"`
	SceneID     string         `bson:"scene_id" json:"scene_id"`
	CharacterID string         `bson:"character_id" json:"character_id"`
	Timestamp   time.Time      `bson:"timestamp" json:"timestamp"`
	Emotion     map[string]any `bson:"emotion,omitempty" json:"emotion,omitempty"`
	Beliefs     []map[string]any `bson:"beliefs,omitempty" json:"beliefs,omitempty"`
	Goal        string         `bson:"goal,omitempty" json:"goal,omitempty"`
	Relationships map[string]float64 `bson:"relationships,omitempty" json:"relationships,omitempty"`
}

type CharacterRepo struct {
	coll        *mongo.Collection
	runtimeColl *mongo.Collection
	stateColl   *mongo.Collection
}

func NewCharacterRepo(client *Client) *CharacterRepo {
	return &CharacterRepo{
		coll:        client.Coll(CollCharacters),
		runtimeColl: client.Coll(CollCharRuntime),
		stateColl:   client.Coll(CollCharSceneState),
	}
}

func (r *CharacterRepo) Create(ctx context.Context, doc *CharacterDoc) error {
	doc.ID = uuid.New().String()
	doc.CreatedAt = time.Now()
	doc.UpdatedAt = doc.CreatedAt
	_, err := r.coll.InsertOne(ctx, doc)
	return err
}

func (r *CharacterRepo) GetByID(ctx context.Context, id string) (*CharacterDoc, error) {
	var doc CharacterDoc
	err := r.coll.FindOne(ctx, bson.M{"_id": id}).Decode(&doc)
	if err != nil {
		return nil, err
	}
	return &doc, nil
}

func (r *CharacterRepo) List(ctx context.Context) ([]CharacterDoc, error) {
	cursor, err := r.coll.Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	var docs []CharacterDoc
	if err := cursor.All(ctx, &docs); err != nil {
		return nil, err
	}
	return docs, nil
}

func (r *CharacterRepo) UpsertRuntime(ctx context.Context, doc *CharRuntimeDoc) error {
	doc.ID = "runtime_" + doc.CharacterID
	doc.UpdatedAt = time.Now()
	_, err := r.runtimeColl.ReplaceOne(
		ctx, bson.M{"character_id": doc.CharacterID}, doc,
		options.Replace().SetUpsert(true),
	)
	return err
}

func (r *CharacterRepo) GetRuntime(ctx context.Context, charID string) (*CharRuntimeDoc, error) {
	var doc CharRuntimeDoc
	err := r.runtimeColl.FindOne(ctx, bson.M{"character_id": charID}).Decode(&doc)
	if err != nil {
		return nil, err
	}
	return &doc, nil
}

func (r *CharacterRepo) CreateSceneState(ctx context.Context, doc *CharSceneStateDoc) error {
	doc.ID = uuid.New().String()
	doc.Timestamp = time.Now()
	_, err := r.stateColl.InsertOne(ctx, doc)
	return err
}

func (r *CharacterRepo) ListSceneStates(ctx context.Context, sceneID string) ([]CharSceneStateDoc, error) {
	cursor, err := r.stateColl.Find(ctx, bson.M{"scene_id": sceneID})
	if err != nil {
		return nil, err
	}
	var docs []CharSceneStateDoc
	if err := cursor.All(ctx, &docs); err != nil {
		return nil, err
	}
	return docs, nil
}
