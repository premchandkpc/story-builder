package narrative

type AnalysisRequest struct {
	SceneID        string   `json:"sceneId"`
	StoryID        string   `json:"storyId"`
	Title          string   `json:"title"`
	Content        string   `json:"content"`
	CharacterNames []string `json:"characterNames"`
}

type SceneAnalysis struct {
	ID                string                `json:"id"`
	SceneID           string                `json:"sceneId"`
	StoryID           string                `json:"storyId"`
	Title             string                `json:"title"`
	Readability       ReadabilityMetrics    `json:"readability"`
	Sentiment         SentimentResult       `json:"sentiment"`
	Pacing            PacingMetrics         `json:"pacing"`
	CharacterMentions []CharacterMention    `json:"characterMentions"`
}

type ReadabilityMetrics struct {
	FleschReadingEase    float64 `json:"fleschReadingEase"`
	FleschKincaidGrade   float64 `json:"fleschKincaidGrade"`
	ColemanLiauIndex     float64 `json:"colemanLiauIndex"`
	AverageSentenceLen   float64 `json:"averageSentenceLength"`
	AverageWordLen       float64 `json:"averageWordLength"`
	LexicalDiversity     float64 `json:"lexicalDiversity"`
	WordCount            int     `json:"wordCount"`
	SentenceCount        int     `json:"sentenceCount"`
	SyllableCount        int     `json:"syllableCount"`
	CharacterCount       int     `json:"characterCount"`
}

type SentimentResult struct {
	OverallScore   float64   `json:"overallScore"`
	OverallLabel   string    `json:"overallLabel"`
	SentenceScores []float64 `json:"sentenceScores"`
	SentenceLabels []string  `json:"sentenceLabels"`
	PositiveRatio  float64   `json:"positiveRatio"`
	NegativeRatio  float64   `json:"negativeRatio"`
	NeutralRatio   float64   `json:"neutralRatio"`
}

type PacingMetrics struct {
	DialogueRatio         float64 `json:"dialogueRatio"`
	NarrativeRatio        float64 `json:"narrativeRatio"`
	ParagraphCount        int     `json:"paragraphCount"`
	AvgParagraphLen       float64 `json:"averageParagraphLength"`
	SentenceVariety       float64 `json:"sentenceVariety"`
}

type CharacterMention struct {
	CharacterName string  `json:"characterName"`
	MentionCount  int     `json:"mentionCount"`
	MentionDensity float64 `json:"mentionDensity"`
}

type AnalysisSummary struct {
	StoryID        string  `json:"storyId"`
	SceneCount     int     `json:"sceneCount"`
	AvgReadability float64 `json:"avgReadability"`
	AvgSentiment   float64 `json:"avgSentiment"`
	AvgDialogue    float64 `json:"avgDialogueRatio"`
}
