package agents

import (
	"context"
	"sync"
	"time"
)

type CharacterEventType string

const (
	EventSceneStart      CharacterEventType = "scene_start"
	EventSceneContext    CharacterEventType = "scene_context"
	EventTurnComplete    CharacterEventType = "turn_complete"
	EventCharAction      CharacterEventType = "character_action"
	EventNarratorOutput  CharacterEventType = "narrator_output"
	EventDirectorCall    CharacterEventType = "director_call"
	EventSceneEnd        CharacterEventType = "scene_end"
	EventStateUpdate     CharacterEventType = "state_update"
	EventQueryIntent     CharacterEventType = "query_intent"
)

type CharacterEvent struct {
	Type      CharacterEventType
	StoryID   string
	SceneID   string
	TurnID    string
	Data      map[string]any
	Timestamp time.Time
}

type CharacterProposal struct {
	CharacterID   string
	ActionType    string
	Content       string
	Priority      int
	TargetCharID  string
}

type CharacterAgentState struct {
	mu               sync.RWMutex
	CharacterID      string
	StoryID          string
	Name             string
	CurrentEmotion   string
	CurrentMood      string
	ActiveGoal       string
	SubGoals         []string
	Knowledge        []string
	KnowledgeGaps    []string
	InternalThoughts []InternalThought
	RecentActions    []ActionRecord
	RelState         map[string]*RelState
	Plan             *ActionPlan
	RecentDialogue   []string
}

type InternalThought struct {
	Timestamp time.Time
	Thought   string
	Type      string
}

type ActionRecord struct {
	SceneID    string
	Turn       int
	ActionType string
	Content    string
}

type RelState struct {
	CharacterID string
	Trust       float64
	Respect     float64
	Fear        float64
	Affection   float64
	Summary     string
}

type ActionPlan struct {
	Goal     string
	Steps    []string
	Priority int
	Active   bool
	FormedAt time.Time
}

func (s *CharacterAgentState) Lock()    { s.mu.Lock() }
func (s *CharacterAgentState) Unlock()  { s.mu.Unlock() }

func (s *CharacterAgentState) RecordThought(thought string, t string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.InternalThoughts = append(s.InternalThoughts, InternalThought{
		Timestamp: time.Now(),
		Thought:   thought,
		Type:      t,
	})
	if len(s.InternalThoughts) > 50 {
		s.InternalThoughts = s.InternalThoughts[len(s.InternalThoughts)-50:]
	}
}

func (s *CharacterAgentState) RecordAction(sceneID string, turn int, atype, content string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.RecentActions = append(s.RecentActions, ActionRecord{
		SceneID: sceneID, Turn: turn, ActionType: atype, Content: content,
	})
	if len(s.RecentActions) > 20 {
		s.RecentActions = s.RecentActions[len(s.RecentActions)-20:]
	}
}

func (s *CharacterAgentState) RecordDialogue(line string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.RecentDialogue = append(s.RecentDialogue, line)
	if len(s.RecentDialogue) > 10 {
		s.RecentDialogue = s.RecentDialogue[len(s.RecentDialogue)-10:]
	}
}

type AgentStateSnapshot struct {
	CharacterID      string            `json:"character_id"`
	Name             string            `json:"name"`
	CurrentEmotion   string            `json:"current_emotion,omitempty"`
	CurrentMood      string            `json:"current_mood,omitempty"`
	ActiveGoal       string            `json:"active_goal,omitempty"`
	SubGoals         []string          `json:"sub_goals,omitempty"`
	Knowledge        []string          `json:"knowledge,omitempty"`
	KnowledgeGaps    []string          `json:"knowledge_gaps,omitempty"`
	InternalThoughts []ThoughtSnapshot `json:"internal_thoughts,omitempty"`
	RecentActions    []ActionSnapshot  `json:"recent_actions,omitempty"`
	RecentDialogue   []string          `json:"recent_dialogue,omitempty"`
	Plan             *PlanSnapshot     `json:"plan,omitempty"`
	Running          bool              `json:"running"`
}

type ThoughtSnapshot struct {
	Timestamp string `json:"timestamp"`
	Thought   string `json:"thought"`
	Type      string `json:"type"`
}

type ActionSnapshot struct {
	SceneID    string `json:"scene_id,omitempty"`
	ActionType string `json:"action_type"`
	Content    string `json:"content"`
}

type PlanSnapshot struct {
	Goal     string   `json:"goal"`
	Steps    []string `json:"steps,omitempty"`
	Priority int      `json:"priority"`
	Active   bool     `json:"active"`
}

type ProposalSnapshot struct {
	CharacterID string `json:"character_id"`
	ActionType  string `json:"action_type"`
	Content     string `json:"content"`
	Priority    int    `json:"priority"`
}

type TokenBudgetChecker interface {
	CheckAndConsume(ctx context.Context, storyID, model, agentType string, promptTokens, completionTokens int) error
}
