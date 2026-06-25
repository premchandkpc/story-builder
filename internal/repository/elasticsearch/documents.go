package elasticsearch

import "github.com/premchand/story-builder/internal/domain"

type storyDoc struct {
	ID          string   `json:"id"`
	Title       string   `json:"title"`
	Description string   `json:"description,omitempty"`
	Theme       string   `json:"theme,omitempty"`
	Genre       string   `json:"genre,omitempty"`
	Status      string   `json:"status"`
	CreatedAt   string   `json:"createdAt"`
	UpdatedAt   string   `json:"updatedAt"`
}

func toStoryDoc(s *domain.Story) storyDoc {
	desc := s.Theme
	if s.MainPrompt != "" {
		desc = s.MainPrompt
	}
	return storyDoc{
		ID:          s.ID,
		Title:       s.Title,
		Description: desc,
		Theme:       s.Theme,
		Genre:       s.Genre,
		Status:      s.Status,
		CreatedAt:   s.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"),
		UpdatedAt:   s.UpdatedAt.UTC().Format("2006-01-02T15:04:05Z"),
	}
}

type sceneDoc struct {
	ID               string   `json:"id"`
	StoryID          string   `json:"storyId"`
	Title            string   `json:"title,omitempty"`
	BeatIntent       string   `json:"beatIntent,omitempty"`
	Content          string   `json:"content,omitempty"`
	Summary          string   `json:"summary,omitempty"`
	Status           string   `json:"status"`
	POV              string   `json:"pov,omitempty"`
	LocationRef      string   `json:"locationRef,omitempty"`
	Participants     []string `json:"participants,omitempty"`
	TimelinePosition int      `json:"timelinePosition,omitempty"`
	TargetWords      int      `json:"targetWords,omitempty"`
	CreatedAt        string   `json:"createdAt"`
	UpdatedAt        string   `json:"updatedAt"`
}

func toSceneDoc(s *domain.Scene) sceneDoc {
	return sceneDoc{
		ID:               s.ID,
		StoryID:          s.StoryID,
		Title:            s.Title,
		BeatIntent:       s.BeatIntent,
		Content:          s.GeneratedContent,
		Summary:          s.Summary,
		Status:           s.Status,
		POV:              s.POV,
		LocationRef:      s.LocationRef,
		Participants:     s.Participants,
		TimelinePosition: s.TimelinePosition,
		TargetWords:      s.TargetWords,
		CreatedAt:        s.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"),
		UpdatedAt:        s.UpdatedAt.UTC().Format("2006-01-02T15:04:05Z"),
	}
}

type characterDoc struct {
	ID        string   `json:"id"`
	StoryID   string   `json:"storyId"`
	CharID    string   `json:"charId"`
	Name      string   `json:"name"`
	Persona   string   `json:"persona,omitempty"`
	Backstory string   `json:"backstory,omitempty"`
	Goals     []string `json:"goals,omitempty"`
	Flaws     []string `json:"flaws,omitempty"`
	ArcType   string   `json:"arcType,omitempty"`
	CreatedAt string   `json:"createdAt"`
}

func toCharacterDoc(c *domain.Character) characterDoc {
	return characterDoc{
		ID:        c.ID,
		StoryID:   c.StoryID,
		CharID:    c.CharID,
		Name:      c.Name,
		Persona:   c.Persona,
		Backstory: c.Backstory,
		Goals:     c.Goals,
		Flaws:     c.Flaws,
		ArcType:   c.ArcType,
		CreatedAt: c.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"),
	}
}
