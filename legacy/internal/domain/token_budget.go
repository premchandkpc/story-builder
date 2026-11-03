package domain

import "time"

type TokenBudget struct {
	ID          string    `bson:"_id" json:"id"`
	StoryID     string    `bson:"storyId" json:"storyId"`
	Model       string    `bson:"model" json:"model"`
	AgentType   string    `bson:"agentType" json:"agentType"`
	PromptTokens     int  `bson:"promptTokens" json:"promptTokens"`
	CompletionTokens int  `bson:"completionTokens" json:"completionTokens"`
	TotalTokens      int  `bson:"totalTokens" json:"totalTokens"`
	TurnCount        int  `bson:"turnCount" json:"turnCount"`
	BudgetLimit      int  `bson:"budgetLimit" json:"budgetLimit"`
	BudgetUsed       int  `bson:"budgetUsed" json:"budgetUsed"`
	CreatedAt  time.Time `bson:"createdAt" json:"createdAt"`
	UpdatedAt  time.Time `bson:"updatedAt,omitempty" json:"updatedAt,omitempty"`
}

type LlmMetrics struct {
	TotalPromptTokens     int                            `json:"total_prompt_tokens"`
	TotalCompletionTokens int                            `json:"total_completion_tokens"`
	TotalTokens           int                            `json:"total_tokens"`
	TotalCostEstimate     float64                        `json:"total_cost_estimate"`
	TurnCount             int                            `json:"turn_count"`
	GenerationCount       int                            `json:"generation_count"`
	ByModel               map[string]ModelTokenUsage     `json:"by_model"`
	ByAgent               map[string]AgentTokenUsage     `json:"by_agent"`
}

type ModelTokenUsage struct {
	PromptTokens     int     `json:"prompt_tokens"`
	CompletionTokens int     `json:"completion_tokens"`
	Cost             float64 `json:"cost"`
}

type AgentTokenUsage struct {
	PromptTokens int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TurnCount    int `json:"turn_count"`
}
