package mongo

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/premchand/story-builder/internal/story"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Repository struct{ db *mongo.Database }

func Connect(ctx context.Context, uri, database string) (*mongo.Client, *Repository, error) {
	connectCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	client, err := mongo.Connect(connectCtx, options.Client().ApplyURI(uri))
	if err != nil {
		return nil, nil, fmt.Errorf("connect MongoDB: %w", err)
	}
	if err := client.Ping(connectCtx, nil); err != nil {
		_ = client.Disconnect(context.Background())
		return nil, nil, fmt.Errorf("ping MongoDB: %w", err)
	}
	repo := &Repository{db: client.Database(database)}
	if err := repo.EnsureIndexes(ctx); err != nil {
		_ = client.Disconnect(context.Background())
		return nil, nil, err
	}
	return client, repo, nil
}

func (r *Repository) EnsureIndexes(ctx context.Context) error {
	indexes := []struct {
		collection string
		model      mongo.IndexModel
	}{
		{"scenes", mongo.IndexModel{Keys: bson.D{{Key: "storyId", Value: 1}}}},
		{"edges", mongo.IndexModel{Keys: bson.D{{Key: "storyId", Value: 1}, {Key: "fromSceneId", Value: 1}, {Key: "toSceneId", Value: 1}}, Options: options.Index().SetUnique(true)}},
		{"drafts", mongo.IndexModel{Keys: bson.D{{Key: "expiresAt", Value: 1}}, Options: options.Index().SetExpireAfterSeconds(0)}},
		{"drafts", mongo.IndexModel{Keys: bson.D{{Key: "sceneId", Value: 1}, {Key: "contextHash", Value: 1}}}},
		{"characters", mongo.IndexModel{Keys: bson.D{{Key: "characterId", Value: 1}, {Key: "version", Value: 1}}, Options: options.Index().SetUnique(true)}},
		{"canon_events", mongo.IndexModel{Keys: bson.D{{Key: "storyId", Value: 1}, {Key: "seq", Value: 1}}, Options: options.Index().SetUnique(true)}},
	}
	for _, item := range indexes {
		if _, err := r.db.Collection(item.collection).Indexes().CreateOne(ctx, item.model); err != nil {
			return fmt.Errorf("ensure %s index: %w", item.collection, err)
		}
	}
	return nil
}

func (r *Repository) CreateStory(ctx context.Context, value story.Story) error {
	_, err := r.db.Collection("stories").InsertOne(ctx, value)
	return err
}
func (r *Repository) GetStory(ctx context.Context, id string) (story.Story, error) {
	var value story.Story
	err := r.db.Collection("stories").FindOne(ctx, bson.M{"_id": id}).Decode(&value)
	return value, findError(err)
}
func (r *Repository) ListStories(ctx context.Context) ([]story.Story, error) {
	return list[story.Story](ctx, r.db.Collection("stories"), bson.M{}, options.Find().SetSort(bson.D{{Key: "createdAt", Value: -1}}))
}
func (r *Repository) CreateScene(ctx context.Context, value story.Scene) error {
	_, err := r.db.Collection("scenes").InsertOne(ctx, value)
	return err
}
func (r *Repository) GetScene(ctx context.Context, id string) (story.Scene, error) {
	var value story.Scene
	err := r.db.Collection("scenes").FindOne(ctx, bson.M{"_id": id}).Decode(&value)
	return value, findError(err)
}
func (r *Repository) UpdateScene(ctx context.Context, value story.Scene) error {
	result, err := r.db.Collection("scenes").ReplaceOne(ctx, bson.M{"_id": value.ID}, value)
	if err != nil {
		return err
	}
	if result.MatchedCount == 0 {
		return story.ErrNotFound
	}
	return nil
}
func (r *Repository) ListScenes(ctx context.Context, storyID string) ([]story.Scene, error) {
	return list[story.Scene](ctx, r.db.Collection("scenes"), bson.M{"storyId": storyID}, options.Find().SetSort(bson.D{{Key: "createdAt", Value: 1}}))
}
func (r *Repository) CreateEdge(ctx context.Context, value story.Edge) error {
	_, err := r.db.Collection("edges").InsertOne(ctx, value)
	if mongo.IsDuplicateKeyError(err) {
		return story.ErrDuplicateEdge
	}
	return err
}
func (r *Repository) ListEdges(ctx context.Context, storyID string) ([]story.Edge, error) {
	return list[story.Edge](ctx, r.db.Collection("edges"), bson.M{"storyId": storyID}, options.Find().SetSort(bson.D{{Key: "createdAt", Value: 1}}))
}
func (r *Repository) CreateDraft(ctx context.Context, value story.Draft) error {
	_, err := r.db.Collection("drafts").InsertOne(ctx, value)
	return err
}
func (r *Repository) FindDraft(ctx context.Context, sceneID, hash string) (story.Draft, error) {
	var value story.Draft
	err := r.db.Collection("drafts").FindOne(ctx, bson.M{"sceneId": sceneID, "contextHash": hash}).Decode(&value)
	return value, findError(err)
}
func (r *Repository) GetDraft(ctx context.Context, id string) (story.Draft, error) {
	var value story.Draft
	err := r.db.Collection("drafts").FindOne(ctx, bson.M{"_id": id}).Decode(&value)
	return value, findError(err)
}
func (r *Repository) UpdateDraft(ctx context.Context, value story.Draft) error {
	result, err := r.db.Collection("drafts").ReplaceOne(ctx, bson.M{"_id": value.ID}, value)
	if err != nil {
		return err
	}
	if result.MatchedCount == 0 {
		return story.ErrNotFound
	}
	return nil
}
func (r *Repository) DeleteDraft(ctx context.Context, id string) error {
	result, err := r.db.Collection("drafts").DeleteOne(ctx, bson.M{"_id": id})
	if err != nil {
		return err
	}
	if result.DeletedCount == 0 {
		return story.ErrNotFound
	}
	return nil
}
func (r *Repository) AcceptDraft(ctx context.Context, id string) (story.Scene, error) {
	var accepted story.Scene
	session, err := r.db.Client().StartSession()
	if err != nil {
		return accepted, err
	}
	defer session.EndSession(ctx)
	_, err = session.WithTransaction(ctx, func(sc mongo.SessionContext) (any, error) {
		draft, err := r.GetDraft(sc, id)
		if err != nil {
			return nil, err
		}
		var scene story.Scene
		if err := r.db.Collection("scenes").FindOne(sc, bson.M{"_id": draft.SceneID}).Decode(&scene); err != nil {
			return nil, findError(err)
		}
		if draft.Status == story.DraftAccepted {
			accepted = scene
			return nil, nil
		}
		if draft.Status != story.DraftReady {
			return nil, story.ErrDraftNotReady
		}
		var current story.Story
		if err := r.db.Collection("stories").FindOne(sc, bson.M{"_id": draft.StoryID}).Decode(&current); err != nil {
			return nil, findError(err)
		}
		if current.CanonVersion != draft.BaseCanonVersion {
			return nil, story.ErrStaleDraft
		}
		scene.AcceptedDraftID = draft.ID
		scene.Prose = draft.Prose
		if _, err := r.db.Collection("scenes").ReplaceOne(sc, bson.M{"_id": scene.ID}, scene); err != nil {
			return nil, err
		}
		increment := int64(len(draft.ProposedEvents))
		if increment == 0 {
			increment = 1
		}
		for i, proposal := range draft.ProposedEvents {
			event := story.CanonEvent{ID: draft.ID + proposal.CharacterID, StoryID: draft.StoryID, Seq: current.CanonVersion + int64(i) + 1, SceneID: draft.SceneID, DraftID: draft.ID, ProposedEvent: proposal, At: draft.CreatedAt}
			if _, err := r.db.Collection("canon_events").InsertOne(sc, event); err != nil {
				return nil, err
			}
		}
		if _, err := r.db.Collection("stories").UpdateOne(sc, bson.M{"_id": current.ID, "canonVersion": current.CanonVersion}, bson.M{"$inc": bson.M{"canonVersion": increment}}); err != nil {
			return nil, err
		}
		if _, err := r.db.Collection("drafts").UpdateOne(sc, bson.M{"_id": draft.ID}, bson.M{"$set": bson.M{"status": story.DraftAccepted}}); err != nil {
			return nil, err
		}
		accepted = scene
		return nil, nil
	})
	return accepted, err
}
func (r *Repository) CreateCharacter(ctx context.Context, value story.Character) error {
	_, err := r.db.Collection("characters").InsertOne(ctx, value)
	return err
}
func (r *Repository) ListCharacters(ctx context.Context, storyID string) ([]story.Character, error) {
	return list[story.Character](ctx, r.db.Collection("characters"), bson.M{"storyId": storyID}, options.Find())
}
func (r *Repository) ListCanonEvents(ctx context.Context, storyID string) ([]story.CanonEvent, error) {
	return list[story.CanonEvent](ctx, r.db.Collection("canon_events"), bson.M{"storyId": storyID}, options.Find().SetSort(bson.D{{Key: "seq", Value: 1}}))
}
func (r *Repository) ListCharacterStates(ctx context.Context, storyID string) ([]story.CharacterState, error) {
	events, err := r.ListCanonEvents(ctx, storyID)
	if err != nil {
		return nil, err
	}
	return story.ProjectCharacterStates(events), nil
}

func findError(err error) error {
	if errors.Is(err, mongo.ErrNoDocuments) {
		return story.ErrNotFound
	}
	return err
}
func list[T any](ctx context.Context, collection *mongo.Collection, filter any, options *options.FindOptions) ([]T, error) {
	cursor, err := collection.Find(ctx, filter, options)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	var values []T
	if err := cursor.All(ctx, &values); err != nil {
		return nil, err
	}
	return values, nil
}
