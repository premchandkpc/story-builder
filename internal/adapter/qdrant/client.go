package qdrant

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	pb "github.com/qdrant/go-client/qdrant"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const (
	CollCharMemory    = "character_memory"
	CollStoryMemory   = "story_memory"
	CollRelMemory     = "relationship_memory"
	CollWorldMemory   = "world_memory"
	CollSceneMemory   = "scene_memory"
	CollLoreMemory    = "lore_memory"
)

type MemoryVector struct {
	ID        uuid.UUID   `json:"id"`
	Payload   map[string]string `json:"payload"`
	Vector    []float32   `json:"vector"`
	Score     float64     `json:"score,omitempty"`
}

type Client struct {
	points pb.PointsClient
	collections pb.CollectionsClient
}

func NewClient(ctx context.Context, addr string) (*Client, error) {
	conn, err := grpc.DialContext(ctx, addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, fmt.Errorf("qdrant connect: %w", err)
	}
	return &Client{
		points:      pb.NewPointsClient(conn),
		collections: pb.NewCollectionsClient(conn),
	}, nil
}

func (c *Client) UpsertVector(ctx context.Context, collection string, id uuid.UUID, vector []float32, payload map[string]string) error {
	pts := []*pb.PointStruct{{
		Id:      &pb.PointId{PointIdOptions: &pb.PointId_Uuid{Uuid: id.String()}},
		Payload: payloadToProto(payload),
		Vectors: &pb.Vectors{VectorsOptions: &pb.Vectors_Vector{Vector: &pb.Vector{Data: vector}}},
	}}
	_, err := c.points.Upsert(ctx, &pb.UpsertPoints{
		CollectionName: collection,
		Points:         pts,
	})
	return err
}

func (c *Client) Search(ctx context.Context, collection string, vector []float32, limit uint64) ([]MemoryVector, error) {
	resp, err := c.points.Search(ctx, &pb.SearchPoints{
		CollectionName: collection,
		Vector:         vector,
		Limit:          limit,
		WithPayload:    &pb.WithPayloadSelector{SelectorOptions: &pb.WithPayloadSelector_Enable{Enable: true}},
	})
	if err != nil {
		return nil, err
	}
	result := make([]MemoryVector, 0, len(resp.Result))
	for _, r := range resp.Result {
		id, _ := uuid.Parse(r.GetId().GetUuid())
		mv := MemoryVector{
			ID:      id,
			Score:   float64(r.GetScore()),
			Payload: protoToPayload(r.GetPayload()),
		}
		if vec := r.GetVectors().GetVector(); vec != nil {
			mv.Vector = vec.GetData()
		}
		result = append(result, mv)
	}
	return result, nil
}

func (c *Client) DeleteVector(ctx context.Context, collection string, id uuid.UUID) error {
	_, err := c.points.Delete(ctx, &pb.DeletePoints{
		CollectionName: collection,
		Points: &pb.PointsSelector{
			PointsSelectorOneOf: &pb.PointsSelector_Points{
				Points: &pb.PointsIdsList{
					Ids: []*pb.PointId{{PointIdOptions: &pb.PointId_Uuid{Uuid: id.String()}}},
				},
			},
		},
	})
	return err
}

func payloadToProto(payload map[string]string) map[string]*pb.Value {
	proto := make(map[string]*pb.Value, len(payload))
	for k, v := range payload {
		proto[k] = &pb.Value{Kind: &pb.Value_StringValue{StringValue: v}}
	}
	return proto
}

func protoToPayload(proto map[string]*pb.Value) map[string]string {
	payload := make(map[string]string, len(proto))
	for k, v := range proto {
		if sv := v.GetStringValue(); sv != "" {
			payload[k] = sv
		}
	}
	return payload
}
