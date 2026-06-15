package mongo

import (
	"context"
	"time"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type SceneDoc struct {
	ID              string         `bson:"_id" json:"id"`
	StoryID         string         `bson:"story_id" json:"story_id"`
	ChapterID       string         `bson:"chapter_id" json:"chapter_id"`
	ParentSceneID   string         `bson:"parent_scene_id,omitempty" json:"parent_scene_id,omitempty"`
	TemplateID      string         `bson:"template_id,omitempty" json:"template_id,omitempty"`
	Title           string         `bson:"title" json:"title"`
	Goal            string         `bson:"goal,omitempty" json:"goal,omitempty"`
	Conflict        string         `bson:"conflict,omitempty" json:"conflict,omitempty"`
	Outcome         string         `bson:"outcome,omitempty" json:"outcome,omitempty"`
	LocationID      string         `bson:"location_id,omitempty" json:"location_id,omitempty"`
	TimelinePos     string         `bson:"timeline_position,omitempty" json:"timeline_position,omitempty"`
	BeatIntent      string         `bson:"beat_intent,omitempty" json:"beat_intent,omitempty"`
	Chars           []string       `bson:"characters" json:"characters"`
	PromptLayers    map[string]any `bson:"prompt_layers,omitempty" json:"prompt_layers,omitempty"`
	Status          string         `bson:"status" json:"status"`
	ParallelGroup   string         `bson:"parallel_group,omitempty" json:"parallel_group,omitempty"`
	CreatedAt       time.Time      `bson:"created_at" json:"created_at"`
	UpdatedAt       time.Time      `bson:"updated_at" json:"updated_at"`
}

type SceneRuntimeDoc struct {
	ID               string              `bson:"_id" json:"id"`
	SceneID          string              `bson:"scene_id" json:"scene_id"`
	WorldState       map[string]any      `bson:"world_state" json:"world_state"`
	CharacterStates  map[string]any      `bson:"character_states" json:"character_states"`
	RelationshipState map[string]any     `bson:"relationship_state" json:"relationship_state"`
	ActiveMemories   []string            `bson:"active_memories" json:"active_memories"`
	Overrides        map[string]any      `bson:"runtime_overrides,omitempty" json:"runtime_overrides,omitempty"`
	UpdatedAt        time.Time           `bson:"updated_at" json:"updated_at"`
}

type ScenePlanDoc struct {
	ID               string         `bson:"_id" json:"id"`
	SceneID          string         `bson:"scene_id" json:"scene_id"`
	Goal             string         `bson:"goal" json:"goal"`
	Conflict         string         `bson:"conflict" json:"conflict"`
	EmotionShift     map[string]string `bson:"emotion_shift" json:"emotion_shift"`
	RelShift         map[string]float64 `bson:"relationship_shift" json:"relationship_shift"`
	ExpectedOutcome  string         `bson:"expected_outcome" json:"expected_outcome"`
	Status           string         `bson:"status" json:"status"`
	CreatedAt        time.Time      `bson:"created_at" json:"created_at"`
}

type SceneRevisionDoc struct {
	ID        string    `bson:"_id" json:"id"`
	SceneID   string    `bson:"scene_id" json:"scene_id"`
	Version   int       `bson:"version" json:"version"`
	Content   string    `bson:"content" json:"content"`
	Editor    string    `bson:"editor,omitempty" json:"editor,omitempty"`
	Comment   string    `bson:"comment,omitempty" json:"comment,omitempty"`
	CreatedAt time.Time `bson:"created_at" json:"created_at"`
}

type SceneBranchDoc struct {
	StoryID     string `bson:"story_id" json:"story_id"`
	FromSceneID string `bson:"from_scene_id" json:"from_scene_id"`
	ToSceneID   string `bson:"to_scene_id" json:"to_scene_id"`
	Condition   string `bson:"condition,omitempty" json:"condition,omitempty"`
	Label       string `bson:"label" json:"label"`
}

type SceneRepo struct {
	coll              *mongo.Collection
	runtimeColl       *mongo.Collection
	planColl          *mongo.Collection
	revisionColl      *mongo.Collection
	branchColl        *mongo.Collection
}

func NewSceneRepo(client *Client) *SceneRepo {
	return &SceneRepo{
		coll:         client.Coll(CollScenes),
		runtimeColl:  client.Coll(CollSceneRuntime),
		planColl:     client.Coll(CollScenePlans),
		revisionColl: client.Coll(CollSceneRevisions),
		branchColl:   client.Coll(CollSceneBranches),
	}
}

func (r *SceneRepo) Create(ctx context.Context, doc *SceneDoc) error {
	doc.ID = uuid.New().String()
	doc.CreatedAt = time.Now()
	doc.UpdatedAt = doc.CreatedAt
	_, err := r.coll.InsertOne(ctx, doc)
	return err
}

func (r *SceneRepo) GetByID(ctx context.Context, id string) (*SceneDoc, error) {
	var doc SceneDoc
	err := r.coll.FindOne(ctx, bson.M{"_id": id}).Decode(&doc)
	if err != nil {
		return nil, err
	}
	return &doc, nil
}

func (r *SceneRepo) ListByChapter(ctx context.Context, chapterID string) ([]SceneDoc, error) {
	cursor, err := r.coll.Find(ctx, bson.M{"chapter_id": chapterID})
	if err != nil {
		return nil, err
	}
	var docs []SceneDoc
	if err := cursor.All(ctx, &docs); err != nil {
		return nil, err
	}
	return docs, nil
}

func (r *SceneRepo) UpsertRuntime(ctx context.Context, doc *SceneRuntimeDoc) error {
	doc.ID = "runtime_" + doc.SceneID
	doc.UpdatedAt = time.Now()
	_, err := r.runtimeColl.ReplaceOne(
		ctx, bson.M{"scene_id": doc.SceneID}, doc,
		options.Replace().SetUpsert(true),
	)
	return err
}

func (r *SceneRepo) GetRuntime(ctx context.Context, sceneID string) (*SceneRuntimeDoc, error) {
	var doc SceneRuntimeDoc
	err := r.runtimeColl.FindOne(ctx, bson.M{"scene_id": sceneID}).Decode(&doc)
	if err != nil {
		return nil, err
	}
	return &doc, nil
}

func (r *SceneRepo) CreatePlan(ctx context.Context, doc *ScenePlanDoc) error {
	doc.ID = uuid.New().String()
	doc.CreatedAt = time.Now()
	_, err := r.planColl.InsertOne(ctx, doc)
	return err
}

func (r *SceneRepo) GetPlan(ctx context.Context, sceneID string) (*ScenePlanDoc, error) {
	var doc ScenePlanDoc
	err := r.planColl.FindOne(ctx, bson.M{"scene_id": sceneID}).Decode(&doc)
	if err != nil {
		return nil, err
	}
	return &doc, nil
}

func (r *SceneRepo) CreateRevision(ctx context.Context, doc *SceneRevisionDoc) error {
	doc.ID = uuid.New().String()
	doc.CreatedAt = time.Now()
	_, err := r.revisionColl.InsertOne(ctx, doc)
	return err
}

func (r *SceneRepo) ListRevisions(ctx context.Context, sceneID string) ([]SceneRevisionDoc, error) {
	cursor, err := r.revisionColl.Find(ctx, bson.M{"scene_id": sceneID})
	if err != nil {
		return nil, err
	}
	var docs []SceneRevisionDoc
	if err := cursor.All(ctx, &docs); err != nil {
		return nil, err
	}
	return docs, nil
}

func (r *SceneRepo) CreateBranch(ctx context.Context, doc *SceneBranchDoc) error {
	_, err := r.branchColl.InsertOne(ctx, doc)
	return err
}

func (r *SceneRepo) ListBranches(ctx context.Context, storyID string) ([]SceneBranchDoc, error) {
	cursor, err := r.branchColl.Find(ctx, bson.M{"story_id": storyID})
	if err != nil {
		return nil, err
	}
	var docs []SceneBranchDoc
	if err := cursor.All(ctx, &docs); err != nil {
		return nil, err
	}
	return docs, nil
}
