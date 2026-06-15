package mongo

import (
	"context"
	"time"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type ChapterDoc struct {
	ID         string    `bson:"_id" json:"id"`
	StoryID    string    `bson:"story_id" json:"story_id"`
	Title      string    `bson:"title" json:"title"`
	Goal       string    `bson:"goal,omitempty" json:"goal,omitempty"`
	Summary    string    `bson:"summary,omitempty" json:"summary,omitempty"`
	OrderIndex int       `bson:"order_index" json:"order_index"`
	Status     string    `bson:"status" json:"status"`
	CreatedAt  time.Time `bson:"created_at" json:"created_at"`
	UpdatedAt  time.Time `bson:"updated_at" json:"updated_at"`
}

type ChapterRepo struct {
	coll *mongo.Collection
}

func NewChapterRepo(client *Client) *ChapterRepo {
	return &ChapterRepo{coll: client.Coll(CollChapters)}
}

func (r *ChapterRepo) Create(ctx context.Context, doc *ChapterDoc) error {
	doc.ID = uuid.New().String()
	doc.CreatedAt = time.Now()
	doc.UpdatedAt = doc.CreatedAt
	_, err := r.coll.InsertOne(ctx, doc)
	return err
}

func (r *ChapterRepo) GetByID(ctx context.Context, id string) (*ChapterDoc, error) {
	var doc ChapterDoc
	err := r.coll.FindOne(ctx, bson.M{"_id": id}).Decode(&doc)
	if err != nil {
		return nil, err
	}
	return &doc, nil
}

func (r *ChapterRepo) ListByStory(ctx context.Context, storyID string) ([]ChapterDoc, error) {
	cursor, err := r.coll.Find(ctx, bson.M{"story_id": storyID})
	if err != nil {
		return nil, err
	}
	var docs []ChapterDoc
	if err := cursor.All(ctx, &docs); err != nil {
		return nil, err
	}
	return docs, nil
}

func (r *ChapterRepo) Update(ctx context.Context, id string, update bson.M) error {
	update["updated_at"] = time.Now()
	_, err := r.coll.UpdateOne(ctx, bson.M{"_id": id}, bson.M{"$set": update})
	return err
}

func (r *ChapterRepo) Delete(ctx context.Context, id string) error {
	_, err := r.coll.DeleteOne(ctx, bson.M{"_id": id})
	return err
}
