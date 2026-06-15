package mongo

import (
	"context"
	"time"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type StoryDoc struct {
	ID            string            `bson:"_id" json:"id"`
	Title         string            `bson:"title" json:"title"`
	Genre         string            `bson:"genre,omitempty" json:"genre,omitempty"`
	Theme         string            `bson:"theme,omitempty" json:"theme,omitempty"`
	Status        string            `bson:"status" json:"status"`
	MainPrompt    string            `bson:"main_prompt,omitempty" json:"main_prompt,omitempty"`
	GeneralPrompt string            `bson:"general_prompt,omitempty" json:"general_prompt,omitempty"`
	CanonPins     map[string]string `bson:"canon_pins,omitempty" json:"canon_pins,omitempty"`
	CreatedAt     time.Time         `bson:"created_at" json:"created_at"`
	UpdatedAt     time.Time         `bson:"updated_at" json:"updated_at"`
}

type StoryRepo struct {
	coll *mongo.Collection
}

func NewStoryRepo(client *Client) *StoryRepo {
	return &StoryRepo{coll: client.Coll(CollStories)}
}

func (r *StoryRepo) Create(ctx context.Context, doc *StoryDoc) error {
	doc.ID = uuid.New().String()
	doc.CreatedAt = time.Now()
	doc.UpdatedAt = doc.CreatedAt
	_, err := r.coll.InsertOne(ctx, doc)
	return err
}

func (r *StoryRepo) GetByID(ctx context.Context, id string) (*StoryDoc, error) {
	var doc StoryDoc
	err := r.coll.FindOne(ctx, bson.M{"_id": id}).Decode(&doc)
	if err != nil {
		return nil, err
	}
	return &doc, nil
}

func (r *StoryRepo) Update(ctx context.Context, id string, update bson.M) error {
	update["updated_at"] = time.Now()
	_, err := r.coll.UpdateOne(ctx, bson.M{"_id": id}, bson.M{"$set": update})
	return err
}

func (r *StoryRepo) Delete(ctx context.Context, id string) error {
	_, err := r.coll.DeleteOne(ctx, bson.M{"_id": id})
	return err
}

func (r *StoryRepo) List(ctx context.Context, filter bson.M, limit, offset int64) ([]StoryDoc, error) {
	opts := options.Find().SetLimit(limit).SetSkip(offset).SetSort(bson.M{"created_at": -1})
	cursor, err := r.coll.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	var docs []StoryDoc
	if err := cursor.All(ctx, &docs); err != nil {
		return nil, err
	}
	return docs, nil
}
