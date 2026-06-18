package domain

import "time"

type Location struct {
	ID          string    `bson:"_id" json:"id"`
	StoryID     string    `bson:"storyId" json:"storyId"`
	Name        string    `bson:"name" json:"name"`
	Description string    `bson:"description,omitempty" json:"description,omitempty"`
	Props       []string  `bson:"props,omitempty" json:"props,omitempty"`
	CreatedAt   time.Time `bson:"createdAt" json:"createdAt"`
}
