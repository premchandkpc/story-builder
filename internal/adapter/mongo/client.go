package mongo

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	CollStories          = "stories"
	CollChapters         = "chapters"
	CollScenes           = "scenes"
	CollSceneTemplates   = "scene_templates"
	CollSceneRuntime     = "scene_runtime"
	CollScenePlans       = "scene_plans"
	CollSceneRevisions   = "scene_revisions"
	CollSceneBranches    = "scene_branches"
	CollCharacters       = "characters"
	CollCharRuntime      = "character_runtime"
	CollCharSceneState   = "character_scene_state"
	CollRelationships    = "relationships"
	CollRelHistory       = "relationship_history"
	CollStoryBibles      = "story_bibles"
	CollTimelineEvents   = "timeline_events"
	CollPromptLayers     = "prompt_layers"
	CollPromptVersions   = "prompt_versions"
	CollLocalizations    = "localization_overlays"
	CollRenderMetadata   = "render_metadata"
	CollRuntimeSnapshots = "runtime_snapshots"
	CollEventStore       = "event_store"
	CollWorkflowState    = "workflow_state"
	CollSagaState        = "saga_state"
	CollGenerationJobs   = "generation_jobs"
	CollGenMetrics       = "generation_metrics"
	CollGenCosts         = "generation_costs"
	CollAgentDecisions   = "agent_decisions"
	CollDirectorDecisions = "director_decisions"
	CollCanonViolations  = "canon_violations"
	CollAssets           = "assets"
	CollUsers            = "users"
	CollOrganizations    = "organizations"
	CollAPIKeys          = "api_keys"
)

type Client struct {
	DB *mongo.Database
}

func NewClient(ctx context.Context, uri, dbName string) (*Client, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		return nil, err
	}
	if err := client.Ping(ctx, nil); err != nil {
		return nil, err
	}
	return &Client{DB: client.Database(dbName)}, nil
}

func (c *Client) Close(ctx context.Context) error {
	return c.DB.Client().Disconnect(ctx)
}

func (c *Client) Coll(name string) *mongo.Collection {
	return c.DB.Collection(name)
}
