package domain

import "time"

type LocationType string

const (
	LocDimension LocationType = "dimension"
	LocPlanet    LocationType = "planet"
	LocCountry   LocationType = "country"
	LocCity      LocationType = "city"
	LocDistrict  LocationType = "district"
	LocBuilding  LocationType = "building"
	LocRoom      LocationType = "room"
)

type Location struct {
	ID          string       `bson:"_id" json:"id"`
	StoryID     string       `bson:"storyId" json:"storyId"`
	Name        string       `bson:"name" json:"name"`
	Description string       `bson:"description,omitempty" json:"description,omitempty"`
	LocType     LocationType `bson:"locType,omitempty" json:"locType,omitempty"`
	ParentID    string       `bson:"parentId,omitempty" json:"parentId,omitempty"`
	Props       []string     `bson:"props,omitempty" json:"props,omitempty"`
	Features    []string     `bson:"features,omitempty" json:"features,omitempty"`
	Atmosphere  string       `bson:"atmosphere,omitempty" json:"atmosphere,omitempty"`
	Children    []string     `bson:"children,omitempty" json:"children,omitempty"`
	CreatedAt   time.Time    `bson:"createdAt" json:"createdAt"`
	UpdatedAt   time.Time    `bson:"updatedAt,omitempty" json:"updatedAt,omitempty"`
}
