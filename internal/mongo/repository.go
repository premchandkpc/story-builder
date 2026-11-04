package mongo

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/premchand/story-builder/internal/mongostore"
	"github.com/premchand/story-builder/internal/story"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readconcern"
	"go.mongodb.org/mongo-driver/mongo/writeconcern"
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
	// Accepted v2 drafts predating this migration must no longer be TTL candidates.
	if _, err := r.db.Collection(collectionDrafts).UpdateMany(ctx, bson.M{fieldStatus: story.DraftAccepted}, bson.M{"$unset": bson.M{fieldExpiresAt: "", "leaseUntil": ""}}); err != nil {
		return fmt.Errorf("retain accepted drafts: %w", err)
	}
	indexes := []struct {
		collection string
		model      mongo.IndexModel
	}{
		{collectionScenes, mongo.IndexModel{Keys: bson.D{{Key: fieldStoryID, Value: 1}}}},
		{collectionEdges, mongo.IndexModel{Keys: bson.D{{Key: fieldStoryID, Value: 1}, {Key: fieldFromSceneID, Value: 1}, {Key: fieldToSceneID, Value: 1}}, Options: options.Index().SetUnique(true)}},
		{collectionDrafts, mongo.IndexModel{Keys: bson.D{{Key: fieldExpiresAt, Value: 1}}, Options: options.Index().SetExpireAfterSeconds(0)}},
		// Only one live generation may own a scene/context. Failed and accepted
		// drafts are deliberately excluded so retries and later canon versions work.
		{collectionDrafts, mongo.IndexModel{Keys: bson.D{{Key: fieldSceneID, Value: 1}, {Key: fieldContextHash, Value: 1}}, Options: options.Index().SetUnique(true).SetPartialFilterExpression(bson.M{fieldStatus: bson.M{"$in": bson.A{story.DraftGenerating, story.DraftReady}}})}},
		{collectionCharacters, mongo.IndexModel{Keys: bson.D{{Key: fieldStoryID, Value: 1}, {Key: fieldVersion, Value: -1}}}},
		{collectionCharacters, mongo.IndexModel{Keys: bson.D{{Key: fieldCharacterID, Value: 1}, {Key: fieldVersion, Value: 1}}, Options: options.Index().SetUnique(true)}},
		{collectionCanonEvents, mongo.IndexModel{Keys: bson.D{{Key: fieldStoryID, Value: 1}, {Key: fieldSequence, Value: 1}}, Options: options.Index().SetUnique(true)}},
	}
	for _, item := range indexes {
		if _, err := r.db.Collection(item.collection).Indexes().CreateOne(ctx, item.model); err != nil {
			return fmt.Errorf("ensure %s index: %w", item.collection, err)
		}
	}
	return nil
}

func (r *Repository) CreateStory(ctx context.Context, value story.Story) error {
	return mongostore.Insert(ctx, r.db.Collection(collectionStories), value)
}
func (r *Repository) GetStory(ctx context.Context, id string) (story.Story, error) {
	return get[story.Story](ctx, r.db.Collection(collectionStories), bson.M{fieldID: id})
}
func (r *Repository) ListStories(ctx context.Context) ([]story.Story, error) {
	return mongostore.List[story.Story](ctx, r.db.Collection(collectionStories), bson.M{}, options.Find().SetSort(bson.D{{Key: fieldCreatedAt, Value: -1}}))
}
func (r *Repository) CreateScene(ctx context.Context, value story.Scene) error {
	return mongostore.Insert(ctx, r.db.Collection(collectionScenes), value)
}
func (r *Repository) GetScene(ctx context.Context, id string) (story.Scene, error) {
	return get[story.Scene](ctx, r.db.Collection(collectionScenes), bson.M{fieldID: id})
}
func (r *Repository) UpdateScene(ctx context.Context, value story.Scene) error {
	filter := bson.M{fieldID: value.ID, "revision": value.Revision - 1}
	if value.Revision == 1 {
		delete(filter, "revision")
		filter["$or"] = bson.A{bson.M{"revision": 0}, bson.M{"revision": bson.M{"$exists": false}}}
	}
	// Accepted scenes may move, but their narrative fields cannot change.
	filter["$and"] = bson.A{bson.M{"$or": bson.A{
		bson.M{"acceptedDraftId": bson.M{"$in": bson.A{"", nil}}},
		bson.M{"title": value.Title, "brief": value.Brief},
	}}}
	result, err := r.db.Collection(collectionScenes).UpdateOne(ctx, filter, bson.M{"$set": bson.M{"title": value.Title, "brief": value.Brief, "position": value.Position, "updatedAt": value.UpdatedAt, "revision": value.Revision, "contentRevision": value.ContentRevision}})
	if err != nil {
		return err
	}
	if result.MatchedCount != 1 {
		latest, err := r.GetScene(ctx, value.ID)
		if err != nil {
			return err
		}
		if latest.AcceptedDraftID != "" && (latest.Title != value.Title || latest.Brief != value.Brief) {
			return story.ErrSceneLocked
		}
		return story.ErrConflict
	}
	return nil
}
func (r *Repository) ListScenes(ctx context.Context, storyID string) ([]story.Scene, error) {
	return mongostore.List[story.Scene](ctx, r.db.Collection(collectionScenes), bson.M{fieldStoryID: storyID}, options.Find().SetSort(bson.D{{Key: fieldCreatedAt, Value: 1}}))
}
func (r *Repository) CreateEdge(ctx context.Context, value story.Edge) error {
	_, err := r.withTransaction(ctx, func(sc mongo.SessionContext) (any, error) {
		target, err := r.GetScene(sc, value.ToSceneID)
		if err != nil {
			return nil, err
		}
		if target.AcceptedDraftID != "" {
			return nil, story.ErrSceneLocked
		}
		edges, err := mongostore.List[story.Edge](sc, r.db.Collection(collectionEdges), bson.M{fieldStoryID: value.StoryID}, options.Find())
		if err != nil {
			return nil, err
		}
		if story.HasCycle(append(edges, value)) {
			return nil, story.ErrCycle
		}
		if _, err := r.db.Collection(collectionEdges).InsertOne(sc, value); err != nil {
			if mongo.IsDuplicateKeyError(err) {
				return nil, story.ErrDuplicateEdge
			}
			return nil, err
		}
		// Every graph mutation writes its story document. Concurrent edge
		// transactions therefore conflict and Mongo retries one against fresh edges.
		result, err := r.db.Collection(collectionStories).UpdateOne(sc, bson.M{fieldID: value.StoryID}, bson.M{"$inc": bson.M{"graphVersion": 1}})
		if err != nil {
			return nil, err
		}
		if result.MatchedCount == 0 {
			return nil, story.ErrNotFound
		}
		return nil, nil
	})
	return err
}
func (r *Repository) ListEdges(ctx context.Context, storyID string) ([]story.Edge, error) {
	return mongostore.List[story.Edge](ctx, r.db.Collection(collectionEdges), bson.M{fieldStoryID: storyID}, options.Find().SetSort(bson.D{{Key: fieldCreatedAt, Value: 1}}))
}
func (r *Repository) CreateDraft(ctx context.Context, value story.Draft) error {
	// TTL cleanup is asynchronous. Delete expired drafts before the unique
	// active-draft insert so an expired READY draft cannot block a retry.
	if _, err := r.db.Collection(collectionDrafts).DeleteMany(ctx, bson.M{fieldStatus: bson.M{"$ne": story.DraftAccepted}, fieldExpiresAt: bson.M{"$lte": time.Now().UTC()}}); err != nil {
		return err
	}
	// A bounded request cannot own generation beyond its lease. Include legacy rows.
	now := time.Now().UTC()
	_, err := r.db.Collection(collectionDrafts).UpdateMany(ctx, bson.M{fieldStatus: story.DraftGenerating, "$or": bson.A{bson.M{"leaseUntil": bson.M{"$lte": now}}, bson.M{"leaseUntil": bson.M{"$exists": false}, fieldCreatedAt: bson.M{"$lte": now.Add(-3 * time.Minute)}}}}, bson.M{"$set": bson.M{fieldStatus: story.DraftFailed}})
	if err != nil {
		return err
	}
	_, err = r.db.Collection(collectionDrafts).InsertOne(ctx, value)
	if mongo.IsDuplicateKeyError(err) {
		return story.ErrDraftInProgress
	}
	return err
}
func (r *Repository) FindDraft(ctx context.Context, sceneID, hash string) (story.Draft, error) {
	return get[story.Draft](ctx, r.db.Collection(collectionDrafts), bson.M{
		fieldSceneID:     sceneID,
		fieldContextHash: hash,
		fieldStatus:      story.DraftReady,
		fieldExpiresAt:   bson.M{"$gt": time.Now().UTC()},
	})
}
func (r *Repository) GetDraft(ctx context.Context, id string) (story.Draft, error) {
	return get[story.Draft](ctx, r.db.Collection(collectionDrafts), bson.M{fieldID: id})
}
func (r *Repository) UpdateDraft(ctx context.Context, value story.Draft) error {
	result, err := r.db.Collection(collectionDrafts).ReplaceOne(ctx, bson.M{fieldID: value.ID, fieldStatus: story.DraftGenerating}, value)
	if err != nil {
		return err
	}
	if result.MatchedCount != 1 {
		return story.ErrDraftNotReady
	}
	return nil
}
func (r *Repository) DeleteDraft(ctx context.Context, id string) error {
	result, err := r.db.Collection(collectionDrafts).DeleteOne(ctx, bson.M{fieldID: id, fieldStatus: bson.M{"$in": bson.A{story.DraftReady, story.DraftFailed}}})
	if err != nil {
		return err
	}
	if result.DeletedCount == 0 {
		return story.ErrDraftNotReady
	}
	return nil
}
func (r *Repository) AcceptDraft(ctx context.Context, id string) (story.Scene, error) {
	var accepted story.Scene
	_, err := r.withTransaction(ctx, func(sc mongo.SessionContext) (any, error) {
		draft, err := r.GetDraft(sc, id)
		if err != nil {
			return nil, err
		}
		var scene story.Scene
		if err := r.db.Collection(collectionScenes).FindOne(sc, bson.M{fieldID: draft.SceneID}).Decode(&scene); err != nil {
			return nil, findError(err)
		}
		if draft.Status == story.DraftAccepted {
			accepted = scene
			return nil, nil
		}
		if draft.Status != story.DraftReady || !draft.ExpiresAt.After(time.Now().UTC()) {
			return nil, story.ErrDraftNotReady
		}
		if err := r.requireAcceptedParents(sc, draft); err != nil {
			return nil, err
		}
		var current story.Story
		if err := r.db.Collection(collectionStories).FindOne(sc, bson.M{fieldID: draft.StoryID}).Decode(&current); err != nil {
			return nil, findError(err)
		}
		if scene.AcceptedDraftID != "" {
			return nil, story.ErrSceneLocked
		}
		if scene.ContentRevision != draft.BaseSceneRevision || current.CharacterRevision != draft.BaseCharacterRevision || current.GraphVersion != draft.BaseGraphVersion || current.CanonVersion != draft.BaseCanonVersion {
			return nil, story.ErrStaleDraft
		}
		scene.AcceptedDraftID = draft.ID
		scene.Prose = draft.Prose
		scene.Revision++
		scene.UpdatedAt = time.Now().UTC()
		if _, err := r.db.Collection(collectionScenes).ReplaceOne(sc, bson.M{fieldID: scene.ID}, scene); err != nil {
			return nil, err
		}
		events := draft.ProposedEvents
		if len(events) == 0 {
			events = []story.ProposedEvent{{Type: "scene_accepted"}}
		}
		for i, proposal := range events {
			event := story.CanonEvent{ID: fmt.Sprintf("%s:%d", draft.ID, i), StoryID: draft.StoryID, Seq: current.CanonVersion + int64(i) + 1, SceneID: draft.SceneID, DraftID: draft.ID, ProposedEvent: proposal, At: time.Now().UTC()}
			if _, err := r.db.Collection(collectionCanonEvents).InsertOne(sc, event); err != nil {
				return nil, err
			}
		}
		increment := int64(len(events))
		result, err := r.db.Collection(collectionStories).UpdateOne(sc, bson.M{fieldID: current.ID, fieldCanonVersion: current.CanonVersion}, bson.M{"$inc": bson.M{fieldCanonVersion: increment}})
		if err != nil {
			return nil, err
		}
		if result.MatchedCount != 1 {
			return nil, story.ErrStaleDraft
		}
		if _, err := r.db.Collection(collectionDrafts).UpdateOne(sc, bson.M{fieldID: draft.ID}, bson.M{"$set": bson.M{fieldStatus: story.DraftAccepted}, "$unset": bson.M{fieldExpiresAt: "", "leaseUntil": ""}}); err != nil {
			return nil, err
		}
		accepted = scene
		return nil, nil
	})
	return accepted, err
}
func (r *Repository) CreateCharacter(ctx context.Context, value story.Character) error {
	return r.SaveCharacter(ctx, value, 0)
}
func (r *Repository) SaveCharacter(ctx context.Context, value story.Character, expected int) error {
	_, err := r.withTransaction(ctx, func(sc mongo.SessionContext) (any, error) {
		var latest story.Character
		e := r.db.Collection(collectionCharacters).FindOne(sc, bson.M{fieldCharacterID: value.CharacterID}, options.FindOne().SetSort(bson.D{{Key: fieldVersion, Value: -1}})).Decode(&latest)
		if e != nil && !errors.Is(e, mongo.ErrNoDocuments) {
			return nil, e
		}
		if (e == nil && (latest.StoryID != value.StoryID || latest.Version != expected)) || (errors.Is(e, mongo.ErrNoDocuments) && expected != 0) || value.Version != expected+1 {
			return nil, story.ErrConflict
		}
		if _, e = r.db.Collection(collectionCharacters).InsertOne(sc, value); e != nil {
			if mongo.IsDuplicateKeyError(e) {
				return nil, story.ErrConflict
			}
			return nil, e
		}
		result, e := r.db.Collection(collectionStories).UpdateOne(sc, bson.M{fieldID: value.StoryID}, bson.M{"$inc": bson.M{"characterRevision": 1}})
		if e != nil {
			return nil, e
		}
		if result.MatchedCount != 1 {
			return nil, story.ErrNotFound
		}
		return nil, nil
	})
	return err
}
func (r *Repository) ListCharacters(ctx context.Context, storyID string) ([]story.Character, error) {
	versions, err := mongostore.List[story.Character](ctx, r.db.Collection(collectionCharacters), bson.M{fieldStoryID: storyID}, options.Find().SetSort(bson.D{{Key: fieldVersion, Value: -1}}))
	if err != nil {
		return nil, err
	}
	values := []story.Character{}
	seen := map[string]bool{}
	for _, value := range versions {
		if !seen[value.CharacterID] {
			values = append(values, value)
			seen[value.CharacterID] = true
		}
	}
	return values, nil
}
func (r *Repository) ListCanonEvents(ctx context.Context, storyID string) ([]story.CanonEvent, error) {
	return mongostore.List[story.CanonEvent](ctx, r.db.Collection(collectionCanonEvents), bson.M{fieldStoryID: storyID}, options.Find().SetSort(bson.D{{Key: fieldSequence, Value: 1}}))
}
func (r *Repository) ListCharacterStates(ctx context.Context, storyID string) ([]story.CharacterState, error) {
	events, err := r.ListCanonEvents(ctx, storyID)
	if err != nil {
		return nil, err
	}
	return story.ProjectCharacterStates(events), nil
}

func (r *Repository) withTransaction(ctx context.Context, fn func(mongo.SessionContext) (any, error)) (any, error) {
	session, err := r.db.Client().StartSession()
	if err != nil {
		return nil, err
	}
	defer session.EndSession(ctx)
	return session.WithTransaction(ctx, fn,
		options.Transaction().SetReadConcern(readconcern.Snapshot()).SetWriteConcern(writeconcern.Majority()),
	)
}

func (r *Repository) requireAcceptedParents(ctx mongo.SessionContext, draft story.Draft) error {
	edges, err := mongostore.List[story.Edge](ctx, r.db.Collection(collectionEdges), bson.M{fieldStoryID: draft.StoryID, fieldToSceneID: draft.SceneID}, nil)
	if err != nil {
		return err
	}
	for _, edge := range edges {
		parent, err := get[story.Scene](ctx, r.db.Collection(collectionScenes), bson.M{fieldID: edge.FromSceneID})
		if err != nil {
			return err
		}
		if parent.AcceptedDraftID == "" {
			return story.ErrParentsPending
		}
	}
	return nil
}

func findError(err error) error {
	if errors.Is(err, mongo.ErrNoDocuments) {
		return story.ErrNotFound
	}
	return err
}
func get[T any](ctx context.Context, collection *mongo.Collection, filter any) (T, error) {
	value, err := mongostore.Get[T](ctx, collection, filter)
	return value, findError(err)
}
func replace[T any](ctx context.Context, collection *mongo.Collection, id string, value T) error {
	matched, err := mongostore.Replace(ctx, collection, bson.M{fieldID: id}, value)
	if err != nil {
		return err
	}
	if !matched {
		return story.ErrNotFound
	}
	return nil
}
