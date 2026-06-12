package server

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/google/uuid"
	"github.com/premchand/story-builder/internal/canon"
	"github.com/premchand/story-builder/internal/compiler"
	"github.com/premchand/story-builder/internal/graph"
	pb "github.com/premchand/story-builder/internal/grpc/gen/storybuilder/v1"
	"github.com/premchand/story-builder/internal/scene"
	canonsvc "github.com/premchand/story-builder/internal/service/canon"
	"github.com/premchand/story-builder/internal/service/edge"
	"github.com/premchand/story-builder/internal/service/generation"
	"github.com/premchand/story-builder/internal/service/node"
	storysvc "github.com/premchand/story-builder/internal/service/story"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"
)

// StoryGenService has a different signature than api.StoryGeneratorService
type StoryGenService interface {
	GenerateStory(ctx context.Context, synopsis string) (storyID string, status string, err error)
}

// ── gRPC service implementations ───────────────────────────────

type characterSrv struct {
	pb.UnimplementedCharacterServiceServer
	svc canonsvc.CharacterService
}

func (s *characterSrv) CreateCharacter(ctx context.Context, req *pb.CreateCharacterRequest) (*pb.Character, error) {
	var pid *uuid.UUID
	if req.ParentId != nil {
		v, err := parseProtoUUID(req.ParentId)
		if err != nil {
			return nil, err
		}
		pid = &v
	}
	r, err := s.svc.Create(ctx, req.Name, req.Persona, req.Backstory, req.MoralAlignment, req.Personality, req.Flaws, req.Goals, req.Traits, req.VoiceSamples, pid, req.Relationships)
	if err != nil {
		return nil, err
	}
	return domainCharToProto(r), nil
}

func (s *characterSrv) GetCharacter(ctx context.Context, req *pb.GetCharacterRequest) (*pb.Character, error) {
	id, err := parseProtoUUID(req.Id)
	if err != nil {
		return nil, err
	}
	r, err := s.svc.Get(ctx, id, int(req.Version))
	if err != nil {
		return nil, err
	}
	return domainCharToProto(r), nil
}

func (s *characterSrv) UpdateCharacter(ctx context.Context, req *pb.UpdateCharacterRequest) (*pb.Character, error) {
	id, err := parseProtoUUID(req.Id)
	if err != nil {
		return nil, err
	}
	var pid *uuid.UUID
	if req.ParentId != nil {
		v, err := parseProtoUUID(req.ParentId)
		if err != nil {
			return nil, err
		}
		pid = &v
	}
	r, err := s.svc.Update(ctx, id, req.Name, req.Persona, req.Backstory, req.MoralAlignment, req.Personality, req.Flaws, req.Goals, req.Traits, req.VoiceSamples, pid, req.Relationships)
	if err != nil {
		return nil, err
	}
	return domainCharToProto(r), nil
}

func (s *characterSrv) ListCharacters(ctx context.Context, _ *pb.Empty) (*pb.ListCharactersResponse, error) {
	list, err := s.svc.List(ctx)
	if err != nil {
		return nil, err
	}
	resp := &pb.ListCharactersResponse{}
	for i := range list {
		resp.Characters = append(resp.Characters, domainCharToProto(&list[i]))
	}
	return resp, nil
}

func domainCharToProto(c *canon.Character) *pb.Character {
	p := &pb.Character{
		Id:             uuidToProto(c.ID),
		Version:        int32(c.Version),
		Name:           c.Name,
		Persona:        c.Persona,
		Backstory:      c.Backstory,
		MoralAlignment: c.MoralAlignment,
		Personality:    c.Personality,
		Flaws:          c.Flaws,
		Goals:          c.Goals,
		Traits:         c.Traits,
		VoiceSamples:   c.VoiceSamples,
		Relationships:  c.Relationships,
		CreatedAt:      timeToProto(c.CreatedAt),
	}
	if c.ParentID != nil {
		p.ParentId = uuidToProto(*c.ParentID)
	}
	return p
}

type actorSrv struct {
	pb.UnimplementedActorServiceServer
	svc canonsvc.ActorService
}

func (s *actorSrv) CreateActor(ctx context.Context, req *pb.CreateActorRequest) (*pb.Actor, error) {
	traits := make(map[string]interface{}, len(req.Traits))
	for k, v := range req.Traits {
		traits[k] = v
	}
	r, err := s.svc.Create(ctx, req.Name, req.Gender, req.Ethnicity, req.Race, req.SkinTone, req.EyeColor, req.HairColor, req.HairStyle, req.Build, req.Nationality, int(req.HeightCm), int(req.WeightKg), int(req.Age), traits)
	if err != nil {
		return nil, err
	}
	return domainActorToProto(r), nil
}

func (s *actorSrv) GetActor(ctx context.Context, req *pb.GetActorRequest) (*pb.Actor, error) {
	id, err := parseProtoUUID(req.Id)
	if err != nil {
		return nil, err
	}
	r, err := s.svc.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	return domainActorToProto(r), nil
}

func (s *actorSrv) UpdateActor(ctx context.Context, req *pb.UpdateActorRequest) (*pb.Actor, error) {
	id, err := parseProtoUUID(req.Id)
	if err != nil {
		return nil, err
	}
	traits := make(map[string]interface{}, len(req.Traits))
	for k, v := range req.Traits {
		traits[k] = v
	}
	r, err := s.svc.Update(ctx, id, req.Name, req.Gender, req.Ethnicity, req.Race, req.SkinTone, req.EyeColor, req.HairColor, req.HairStyle, req.Build, req.Nationality, int(req.HeightCm), int(req.WeightKg), int(req.Age), traits)
	if err != nil {
		return nil, err
	}
	return domainActorToProto(r), nil
}

func (s *actorSrv) ListActors(ctx context.Context, _ *pb.Empty) (*pb.ListActorsResponse, error) {
	list, err := s.svc.List(ctx)
	if err != nil {
		return nil, err
	}
	resp := &pb.ListActorsResponse{}
	for i := range list {
		resp.Actors = append(resp.Actors, domainActorToProto(&list[i]))
	}
	return resp, nil
}

func domainActorToProto(a *canon.Actor) *pb.Actor {
	p := &pb.Actor{
		Id:          uuidToProto(a.ID),
		Name:        a.Name,
		Gender:      a.Gender,
		Ethnicity:   a.Ethnicity,
		Race:        a.Race,
		SkinTone:    a.SkinTone,
		EyeColor:    a.EyeColor,
		HairColor:   a.HairColor,
		HairStyle:   a.HairStyle,
		Build:       a.Build,
		HeightCm:    int32(a.HeightCm),
		WeightKg:    int32(a.WeightKg),
		Age:         int32(a.Age),
		Nationality: a.Nationality,
		Traits:      make(map[string]string, len(a.Traits)),
		CreatedAt:   timeToProto(a.CreatedAt),
	}
	for k, v := range a.Traits {
		p.Traits[k] = fmt.Sprintf("%v", v)
	}
	return p
}

type traitSrv struct {
	pb.UnimplementedCharacterTraitServiceServer
	svc canonsvc.TraitService
}

func (s *traitSrv) CreateTrait(ctx context.Context, req *pb.CreateCharacterTraitRequest) (*pb.CharacterTrait, error) {
	r, err := s.svc.Create(ctx, req.Name, req.Category, req.Description)
	if err != nil {
		return nil, err
	}
	return &pb.CharacterTrait{
		Id:          uuidToProto(r.ID),
		Name:        r.Name,
		Category:    r.Category,
		Description: r.Description,
		CreatedAt:   timeToProto(r.CreatedAt),
	}, nil
}

func (s *traitSrv) GetTrait(ctx context.Context, req *pb.GetCharacterTraitRequest) (*pb.CharacterTrait, error) {
	id, err := parseProtoUUID(req.Id)
	if err != nil {
		return nil, err
	}
	r, err := s.svc.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	return &pb.CharacterTrait{
		Id:          uuidToProto(r.ID),
		Name:        r.Name,
		Category:    r.Category,
		Description: r.Description,
		CreatedAt:   timeToProto(r.CreatedAt),
	}, nil
}

func (s *traitSrv) ListTraits(ctx context.Context, _ *pb.Empty) (*pb.ListCharacterTraitsResponse, error) {
	list, err := s.svc.List(ctx)
	if err != nil {
		return nil, err
	}
	resp := &pb.ListCharacterTraitsResponse{}
	for _, t := range list {
		resp.Traits = append(resp.Traits, &pb.CharacterTrait{
			Id:          uuidToProto(t.ID),
			Name:        t.Name,
			Category:    t.Category,
			Description: t.Description,
			CreatedAt:   timeToProto(t.CreatedAt),
		})
	}
	return resp, nil
}

func (s *traitSrv) AssignTrait(ctx context.Context, req *pb.AssignTraitRequest) (*pb.Empty, error) {
	charID, err := parseProtoUUID(req.CharacterId)
	if err != nil {
		return nil, err
	}
	traitID, err := parseProtoUUID(req.TraitId)
	if err != nil {
		return nil, err
	}
	if err := s.svc.Assign(ctx, charID, traitID, int(req.Intensity), req.Note); err != nil {
		return nil, err
	}
	return &pb.Empty{}, nil
}

func (s *traitSrv) UnassignTrait(ctx context.Context, req *pb.UnassignTraitRequest) (*pb.Empty, error) {
	charID, err := parseProtoUUID(req.CharacterId)
	if err != nil {
		return nil, err
	}
	traitID, err := parseProtoUUID(req.TraitId)
	if err != nil {
		return nil, err
	}
	if err := s.svc.Unassign(ctx, charID, traitID); err != nil {
		return nil, err
	}
	return &pb.Empty{}, nil
}

func (s *traitSrv) GetTraitAssignments(ctx context.Context, req *pb.GetTraitAssignmentsRequest) (*pb.GetTraitAssignmentsResponse, error) {
	charID, err := parseProtoUUID(req.CharacterId)
	if err != nil {
		return nil, err
	}
	list, err := s.svc.GetAssignments(ctx, charID)
	if err != nil {
		return nil, err
	}
	resp := &pb.GetTraitAssignmentsResponse{}
	for _, a := range list {
		resp.Assignments = append(resp.Assignments, &pb.TraitAssignment{
			CharacterId: uuidToProto(a.CharacterID),
			TraitId:     uuidToProto(a.TraitID),
			Intensity:   int32(a.Intensity),
			Note:        a.Note,
		})
	}
	return resp, nil
}

type castingSrv struct {
	pb.UnimplementedCastingServiceServer
	svc canonsvc.CastingService
}

func (s *castingSrv) CreateCasting(ctx context.Context, req *pb.CreateCastingRequest) (*pb.Casting, error) {
	storyID, err := parseProtoUUID(req.StoryId)
	if err != nil {
		return nil, err
	}
	actorID, err := parseProtoUUID(req.ActorId)
	if err != nil {
		return nil, err
	}
	charID, err := parseProtoUUID(req.CharacterId)
	if err != nil {
		return nil, err
	}
	r, err := s.svc.Create(ctx, storyID, actorID, charID, req.RoleType)
	if err != nil {
		return nil, err
	}
	return &pb.Casting{
		Id:          uuidToProto(r.ID),
		StoryId:     uuidToProto(r.StoryID),
		ActorId:     uuidToProto(r.ActorID),
		CharacterId: uuidToProto(r.CharacterID),
		RoleType:    r.RoleType,
		CreatedAt:   timeToProto(r.CreatedAt),
	}, nil
}

func (s *castingSrv) ListCastingForStory(ctx context.Context, req *pb.ListCastingForStoryRequest) (*pb.ListCastingResponse, error) {
	id, err := parseProtoUUID(req.StoryId)
	if err != nil {
		return nil, err
	}
	return listCasting(s.svc.GetForStory(ctx, id))
}

func (s *castingSrv) ListCastingForActor(ctx context.Context, req *pb.ListCastingForActorRequest) (*pb.ListCastingResponse, error) {
	id, err := parseProtoUUID(req.ActorId)
	if err != nil {
		return nil, err
	}
	return listCasting(s.svc.GetForActor(ctx, id))
}

func (s *castingSrv) ListCastingForCharacter(ctx context.Context, req *pb.ListCastingForCharacterRequest) (*pb.ListCastingResponse, error) {
	id, err := parseProtoUUID(req.CharacterId)
	if err != nil {
		return nil, err
	}
	return listCasting(s.svc.GetForCharacter(ctx, id))
}

func listCasting(list []canon.Casting, err error) (*pb.ListCastingResponse, error) {
	if err != nil {
		return nil, err
	}
	resp := &pb.ListCastingResponse{}
	for _, c := range list {
		resp.Castings = append(resp.Castings, &pb.Casting{
			Id:          uuidToProto(c.ID),
			StoryId:     uuidToProto(c.StoryID),
			ActorId:     uuidToProto(c.ActorID),
			CharacterId: uuidToProto(c.CharacterID),
			RoleType:    c.RoleType,
			CreatedAt:   timeToProto(c.CreatedAt),
		})
	}
	return resp, nil
}

type locationSrv struct {
	pb.UnimplementedLocationServiceServer
	svc canonsvc.LocationService
}

func (s *locationSrv) CreateLocation(ctx context.Context, req *pb.CreateLocationRequest) (*pb.Location, error) {
	r, err := s.svc.Create(ctx, req.Name, req.Description, req.Props)
	if err != nil {
		return nil, err
	}
	return domainLocToProto(r), nil
}

func (s *locationSrv) GetLocation(ctx context.Context, req *pb.GetLocationRequest) (*pb.Location, error) {
	id, err := parseProtoUUID(req.Id)
	if err != nil {
		return nil, err
	}
	r, err := s.svc.Get(ctx, id, int(req.Version))
	if err != nil {
		return nil, err
	}
	return domainLocToProto(r), nil
}

func (s *locationSrv) UpdateLocation(ctx context.Context, req *pb.UpdateLocationRequest) (*pb.Location, error) {
	id, err := parseProtoUUID(req.Id)
	if err != nil {
		return nil, err
	}
	r, err := s.svc.Update(ctx, id, req.Description, req.Props)
	if err != nil {
		return nil, err
	}
	return domainLocToProto(r), nil
}

func (s *locationSrv) ListLocations(ctx context.Context, _ *pb.Empty) (*pb.ListLocationsResponse, error) {
	list, err := s.svc.List(ctx)
	if err != nil {
		return nil, err
	}
	resp := &pb.ListLocationsResponse{}
	for i := range list {
		resp.Locations = append(resp.Locations, domainLocToProto(&list[i]))
	}
	return resp, nil
}

func domainLocToProto(l *canon.Location) *pb.Location {
	return &pb.Location{
		Id:          uuidToProto(l.ID),
		Version:     int32(l.Version),
		Name:        l.Name,
		Description: l.Description,
		Props:       l.Props,
		CreatedAt:   timeToProto(l.CreatedAt),
	}
}

type loreSrv struct {
	pb.UnimplementedLoreServiceServer
	svc canonsvc.LoreService
}

func (s *loreSrv) CreateLore(ctx context.Context, req *pb.CreateLoreRequest) (*pb.Lore, error) {
	r, err := s.svc.Create(ctx, req.Tags, req.Content)
	if err != nil {
		return nil, err
	}
	return &pb.Lore{
		Id:        uuidToProto(r.ID),
		Tags:      r.Tags,
		Content:   r.Content,
		CreatedAt: timeToProto(r.CreatedAt),
	}, nil
}

func (s *loreSrv) ListLore(ctx context.Context, _ *pb.Empty) (*pb.ListLoreResponse, error) {
	list, err := s.svc.List(ctx)
	if err != nil {
		return nil, err
	}
	resp := &pb.ListLoreResponse{}
	for _, l := range list {
		resp.Lore = append(resp.Lore, &pb.Lore{
			Id:        uuidToProto(l.ID),
			Tags:      l.Tags,
			Content:   l.Content,
			CreatedAt: timeToProto(l.CreatedAt),
		})
	}
	return resp, nil
}

func (s *loreSrv) SearchByTags(ctx context.Context, req *pb.SearchLoreByTagsRequest) (*pb.ListLoreResponse, error) {
	list, err := s.svc.SearchByTags(ctx, req.Tags)
	if err != nil {
		return nil, err
	}
	resp := &pb.ListLoreResponse{}
	for _, l := range list {
		resp.Lore = append(resp.Lore, &pb.Lore{
			Id:        uuidToProto(l.ID),
			Tags:      l.Tags,
			Content:   l.Content,
			CreatedAt: timeToProto(l.CreatedAt),
		})
	}
	return resp, nil
}

func (s *loreSrv) SearchSimilar(ctx context.Context, req *pb.SearchLoreSimilarRequest) (*pb.ListLoreResponse, error) {
	list, err := s.svc.SearchSimilar(ctx, req.Embedding, int(req.Limit))
	if err != nil {
		return nil, err
	}
	resp := &pb.ListLoreResponse{}
	for _, l := range list {
		resp.Lore = append(resp.Lore, &pb.Lore{
			Id:        uuidToProto(l.ID),
			Tags:      l.Tags,
			Content:   l.Content,
			CreatedAt: timeToProto(l.CreatedAt),
		})
	}
	return resp, nil
}

type storySrv struct {
	pb.UnimplementedStoryServiceServer
	storySvc storysvc.StoryService
	edgeSvc  edge.Service
	nodeSvc  node.Service
}

func (s *storySrv) CreateStory(ctx context.Context, req *pb.CreateStoryRequest) (*pb.Story, error) {
	r, err := s.storySvc.Create(ctx, req.Title)
	if err != nil {
		return nil, err
	}
	return &pb.Story{
		Id:        uuidToProto(r.ID),
		Title:     r.Title,
		CreatedAt: timeToProto(r.CreatedAt),
	}, nil
}

func (s *storySrv) GetStory(ctx context.Context, req *pb.GetStoryRequest) (*pb.Story, error) {
	id, err := parseProtoUUID(req.Id)
	if err != nil {
		return nil, err
	}
	r, err := s.storySvc.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	return &pb.Story{
		Id:        uuidToProto(r.ID),
		Title:     r.Title,
		CreatedAt: timeToProto(r.CreatedAt),
	}, nil
}

func (s *storySrv) ListStories(ctx context.Context, _ *pb.Empty) (*pb.ListStoriesResponse, error) {
	list, err := s.storySvc.List(ctx)
	if err != nil {
		return nil, err
	}
	resp := &pb.ListStoriesResponse{}
	for _, st := range list {
		resp.Stories = append(resp.Stories, &pb.Story{
			Id:        uuidToProto(st.ID),
			Title:     st.Title,
			CreatedAt: timeToProto(st.CreatedAt),
		})
	}
	return resp, nil
}

func (s *storySrv) CreateEdge(ctx context.Context, req *pb.CreateEdgeRequest) (*pb.Empty, error) {
	storyID, err := parseProtoUUID(req.StoryId)
	if err != nil {
		return nil, err
	}
	fromID, err := parseProtoUUID(req.FromNode)
	if err != nil {
		return nil, err
	}
	toID, err := parseProtoUUID(req.ToNode)
	if err != nil {
		return nil, err
	}
	et := protoEdgeTypeToDomain(req.EdgeType)
	if err := s.edgeSvc.Create(ctx, storyID, fromID, toID, string(et)); err != nil {
		return nil, err
	}
	return &pb.Empty{}, nil
}

func (s *storySrv) ListEdges(ctx context.Context, req *pb.ListEdgesRequest) (*pb.ListEdgesResponse, error) {
	id, err := parseProtoUUID(req.StoryId)
	if err != nil {
		return nil, err
	}
	list, err := s.edgeSvc.List(ctx, id)
	if err != nil {
		return nil, err
	}
	resp := &pb.ListEdgesResponse{}
	for _, e := range list {
		resp.Edges = append(resp.Edges, domainEdgeToProto(e))
	}
	return resp, nil
}

func (s *storySrv) GetTopology(ctx context.Context, req *pb.GetTopologyRequest) (*pb.Topology, error) {
	id, err := parseProtoUUID(req.StoryId)
	if err != nil {
		return nil, err
	}
	nodes, err := s.nodeSvc.List(ctx, id)
	if err != nil {
		return nil, err
	}
	edges, err := s.edgeSvc.List(ctx, id)
	if err != nil {
		return nil, err
	}
	resp := &pb.Topology{}
	for _, n := range nodes {
		resp.Nodes = append(resp.Nodes, domainNodeToProto(&n))
	}
	for _, e := range edges {
		resp.Edges = append(resp.Edges, domainEdgeToProto(e))
	}
	return resp, nil
}

func domainEdgeToProto(e graph.Edge) *pb.Edge {
	return &pb.Edge{
		StoryId:  uuidToProto(e.StoryID),
		FromNode: uuidToProto(e.FromNode),
		ToNode:   uuidToProto(e.ToNode),
		EdgeType: domainEdgeTypeToProto(e.EdgeType),
	}
}

func domainEdgeTypeToProto(t graph.EdgeType) pb.EdgeType {
	switch t {
	case graph.EdgeTypeSeq:
		return pb.EdgeType_EDGE_TYPE_SEQ
	case graph.EdgeTypeFork:
		return pb.EdgeType_EDGE_TYPE_FORK
	case graph.EdgeTypeJoin:
		return pb.EdgeType_EDGE_TYPE_JOIN
	case graph.EdgeTypeChoice:
		return pb.EdgeType_EDGE_TYPE_CHOICE
	default:
		return pb.EdgeType_EDGE_TYPE_UNSPECIFIED
	}
}

func protoEdgeTypeToDomain(t pb.EdgeType) graph.EdgeType {
	switch t {
	case pb.EdgeType_EDGE_TYPE_SEQ:
		return graph.EdgeTypeSeq
	case pb.EdgeType_EDGE_TYPE_FORK:
		return graph.EdgeTypeFork
	case pb.EdgeType_EDGE_TYPE_JOIN:
		return graph.EdgeTypeJoin
	case pb.EdgeType_EDGE_TYPE_CHOICE:
		return graph.EdgeTypeChoice
	default:
		return graph.EdgeTypeSeq
	}
}

type nodeSrv struct {
	pb.UnimplementedNodeServiceServer
	svc node.Service
}

func (s *nodeSrv) CreateNode(ctx context.Context, req *pb.CreateNodeRequest) (*pb.Node, error) {
	storyID, err := parseProtoUUID(req.StoryId)
	if err != nil {
		return nil, err
	}
	charRefs := make([]uuid.UUID, len(req.CharacterRefs))
	for i, r := range req.CharacterRefs {
		id, err := parseProtoUUID(r)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid character_ref[%d]: %v", i, err)
		}
		charRefs[i] = id
	}
	var locRef *uuid.UUID
	if req.LocationRef != nil {
		v, err := parseProtoUUID(req.LocationRef)
		if err != nil {
			return nil, err
		}
		locRef = &v
	}
	domainNode, err := s.svc.Create(ctx, storyID, req.BeatIntent, charRefs, locRef, req.Pov, req.Tone, int(req.TargetWords))
	if err != nil {
		return nil, err
	}
	if req.SceneStructure != nil {
		ss := protoSceneStructureToDomain(req.SceneStructure)
		if err := s.svc.SetSceneStructure(ctx, domainNode.ID, ss); err != nil {
			return nil, err
		}
		domainNode.SceneStructure = &ss
	}
	return domainNodeToProto(domainNode), nil
}

func (s *nodeSrv) GetNode(ctx context.Context, req *pb.GetNodeRequest) (*pb.Node, error) {
	id, err := parseProtoUUID(req.Id)
	if err != nil {
		return nil, err
	}
	r, err := s.svc.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	return domainNodeToProto(r), nil
}

func (s *nodeSrv) UpdateNode(ctx context.Context, req *pb.UpdateNodeRequest) (*pb.Node, error) {
	nodeID, err := parseProtoUUID(req.Id)
	if err != nil {
		return nil, err
	}
	charRefs := make([]uuid.UUID, len(req.CharacterRefs))
	for i, r := range req.CharacterRefs {
		id, err := parseProtoUUID(r)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid character_ref[%d]: %v", i, err)
		}
		charRefs[i] = id
	}
	var locRef *uuid.UUID
	if req.LocationRef != nil {
		v, err := parseProtoUUID(req.LocationRef)
		if err != nil {
			return nil, err
		}
		locRef = &v
	}
	var ss *graph.SceneStructure
	if req.SceneStructure != nil {
		v := protoSceneStructureToDomain(req.SceneStructure)
		ss = &v
	}
	r, err := s.svc.Update(ctx, nodeID, req.BeatIntent, charRefs, locRef, req.Pov, req.Tone, int(req.TargetWords), ss)
	if err != nil {
		return nil, err
	}
	return domainNodeToProto(r), nil
}

func (s *nodeSrv) ListNodes(ctx context.Context, req *pb.ListNodesRequest) (*pb.ListNodesResponse, error) {
	id, err := parseProtoUUID(req.StoryId)
	if err != nil {
		return nil, err
	}
	list, err := s.svc.List(ctx, id)
	if err != nil {
		return nil, err
	}
	resp := &pb.ListNodesResponse{}
	for i := range list {
		resp.Nodes = append(resp.Nodes, domainNodeToProto(&list[i]))
	}
	return resp, nil
}

func domainNodeToProto(n *graph.Node) *pb.Node {
	charRefs := make([]*pb.UUID, len(n.CharacterRefs))
	for i, r := range n.CharacterRefs {
		charRefs[i] = uuidToProto(r)
	}
	p := &pb.Node{
		Id:            uuidToProto(n.ID),
		StoryId:       uuidToProto(n.StoryID),
		BeatIntent:    n.BeatIntent,
		CharacterRefs: charRefs,
		Pov:           n.POV,
		Tone:          n.Tone,
		TargetWords:   int32(n.TargetWords),
		Status:        domainNodeStatusToProto(n.Status),
		CreatedAt:     timeToProto(n.CreatedAt),
		UpdatedAt:     timeToProto(n.UpdatedAt),
	}
	if n.LocationRef != nil {
		p.LocationRef = uuidToProto(*n.LocationRef)
	}
	if n.SceneStructure != nil {
		p.SceneStructure = domainSceneStructureToProto(n.SceneStructure)
	}
	return p
}

func domainNodeStatusToProto(s graph.NodeStatus) pb.NodeStatus {
	switch s {
	case graph.NodeStatusDraft:
		return pb.NodeStatus_NODE_STATUS_DRAFT
	case graph.NodeStatusGenerated:
		return pb.NodeStatus_NODE_STATUS_GENERATED
	case graph.NodeStatusAccepted:
		return pb.NodeStatus_NODE_STATUS_ACCEPTED
	case graph.NodeStatusStale:
		return pb.NodeStatus_NODE_STATUS_STALE
	default:
		return pb.NodeStatus_NODE_STATUS_UNSPECIFIED
	}
}

func domainFlowTypeToProto(t graph.FlowType) pb.FlowType {
	switch t {
	case graph.FlowMonologue:
		return pb.FlowType_FLOW_TYPE_MONOLOGUE
	case graph.FlowDialogue:
		return pb.FlowType_FLOW_TYPE_DIALOGUE
	case graph.FlowRoundRobin:
		return pb.FlowType_FLOW_TYPE_ROUND_ROBIN
	case graph.FlowParallel:
		return pb.FlowType_FLOW_TYPE_PARALLEL
	case graph.FlowCustom:
		return pb.FlowType_FLOW_TYPE_CUSTOM
	default:
		return pb.FlowType_FLOW_TYPE_UNSPECIFIED
	}
}

func protoFlowTypeToDomain(t pb.FlowType) graph.FlowType {
	switch t {
	case pb.FlowType_FLOW_TYPE_MONOLOGUE:
		return graph.FlowMonologue
	case pb.FlowType_FLOW_TYPE_DIALOGUE:
		return graph.FlowDialogue
	case pb.FlowType_FLOW_TYPE_ROUND_ROBIN:
		return graph.FlowRoundRobin
	case pb.FlowType_FLOW_TYPE_PARALLEL:
		return graph.FlowParallel
	case pb.FlowType_FLOW_TYPE_CUSTOM:
		return graph.FlowCustom
	default:
		return graph.FlowMonologue
	}
}

func domainSceneStructureToProto(ss *graph.SceneStructure) *pb.SceneStructure {
	charOrder := make([]*pb.UUID, len(ss.CharacterOrder))
	for i, c := range ss.CharacterOrder {
		charOrder[i] = uuidToProto(c)
	}
	return &pb.SceneStructure{
		FlowType:       domainFlowTypeToProto(ss.FlowType),
		CharacterOrder: charOrder,
		SituationFlow:  ss.SituationFlow,
		MaxTurns:       int32(ss.MaxTurns),
	}
}

func protoSceneStructureToDomain(ss *pb.SceneStructure) graph.SceneStructure {
	charOrder := make([]uuid.UUID, len(ss.CharacterOrder))
	for i, c := range ss.CharacterOrder {
		id, err := parseProtoUUID(c)
		if err != nil {
			return graph.SceneStructure{}
		}
		charOrder[i] = id
	}
	return graph.SceneStructure{
		FlowType:       protoFlowTypeToDomain(ss.FlowType),
		CharacterOrder: charOrder,
		SituationFlow:  ss.SituationFlow,
		MaxTurns:       int(ss.MaxTurns),
	}
}

type generationSrv struct {
	pb.UnimplementedGenerationServiceServer
	svc generation.GenerationService
}

func (s *generationSrv) Generate(ctx context.Context, req *pb.GenerateRequest) (*pb.Generation, error) {
	nodeID, err := parseProtoUUID(req.NodeId)
	if err != nil {
		return nil, err
	}
	r, err := s.svc.Generate(ctx, nodeID)
	if err != nil {
		return nil, err
	}
	return domainGenToProto(r), nil
}

func (s *generationSrv) AcceptGeneration(ctx context.Context, req *pb.AcceptGenerationRequest) (*pb.Empty, error) {
	nodeID, err := parseProtoUUID(req.NodeId)
	if err != nil {
		return nil, err
	}
	genID, err := parseProtoUUID(req.GenerationId)
	if err != nil {
		return nil, err
	}
	if err := s.svc.AcceptGeneration(ctx, nodeID, genID); err != nil {
		return nil, err
	}
	return &pb.Empty{}, nil
}

func (s *generationSrv) ListGenerations(ctx context.Context, req *pb.ListGenerationsRequest) (*pb.ListGenerationsResponse, error) {
	nodeID, err := parseProtoUUID(req.NodeId)
	if err != nil {
		return nil, err
	}
	list, err := s.svc.ListGenerations(ctx, nodeID)
	if err != nil {
		return nil, err
	}
	resp := &pb.ListGenerationsResponse{}
	for _, g := range list {
		resp.Generations = append(resp.Generations, domainGenToProto(&g))
	}
	return resp, nil
}

func domainGenToProto(g *compiler.Generation) *pb.Generation {
	nodeID, _ := uuid.Parse(g.NodeID)
	genID, _ := uuid.Parse(g.ID)
	return &pb.Generation{
		Id:             uuidToProto(genID),
		NodeId:         uuidToProto(nodeID),
		ContextHash:    g.ContextHash,
		PromptSnapshot: g.PromptSnapshot,
		Output:         g.Output,
		Model:          g.Model,
		Accepted:       g.Accepted,
	}
}

type sceneSrv struct {
	pb.UnimplementedSceneServiceServer
	svc scene.SceneService
}

func (s *sceneSrv) SetSceneStructure(ctx context.Context, req *pb.SetSceneStructureRequest) (*pb.Empty, error) {
	nodeID, err := parseProtoUUID(req.NodeId)
	if err != nil {
		return nil, err
	}
	if err := s.svc.SetSceneStructure(ctx, nodeID, protoSceneStructureToDomain(req.SceneStructure)); err != nil {
		return nil, err
	}
	return &pb.Empty{}, nil
}

func (s *sceneSrv) GetSceneStructure(ctx context.Context, req *pb.GetSceneStructureRequest) (*pb.SceneStructure, error) {
	nodeID, err := parseProtoUUID(req.NodeId)
	if err != nil {
		return nil, err
	}
	r, err := s.svc.GetSceneStructure(ctx, nodeID)
	if err != nil {
		return nil, err
	}
	if r == nil {
		return nil, fmt.Errorf("scene structure not set for node %s", req.NodeId.Value)
	}
	return domainSceneStructureToProto(r), nil
}

func (s *sceneSrv) StartScene(ctx context.Context, req *pb.StartSceneRequest) (*pb.SceneTurn, error) {
	nodeID, err := parseProtoUUID(req.NodeId)
	if err != nil {
		return nil, err
	}
	r, err := s.svc.StartScene(ctx, nodeID)
	if err != nil {
		return nil, err
	}
	return domainTurnToProto(r), nil
}

func (s *sceneSrv) NextTurn(ctx context.Context, req *pb.NextTurnRequest) (*pb.SceneTurn, error) {
	nodeID, err := parseProtoUUID(req.NodeId)
	if err != nil {
		return nil, err
	}
	r, err := s.svc.NextTurn(ctx, nodeID)
	if err != nil {
		return nil, err
	}
	return domainTurnToProto(r), nil
}

func (s *sceneSrv) FinishScene(ctx context.Context, req *pb.FinishSceneRequest) (*pb.FinishSceneResponse, error) {
	nodeID, err := parseProtoUUID(req.NodeId)
	if err != nil {
		return nil, err
	}
	r, err := s.svc.FinishScene(ctx, nodeID)
	if err != nil {
		return nil, err
	}
	return &pb.FinishSceneResponse{Output: r}, nil
}

func (s *sceneSrv) GetTurns(ctx context.Context, req *pb.GetTurnsRequest) (*pb.ListTurnsResponse, error) {
	nodeID, err := parseProtoUUID(req.NodeId)
	if err != nil {
		return nil, err
	}
	list, err := s.svc.GetTurns(ctx, nodeID)
	if err != nil {
		return nil, err
	}
	resp := &pb.ListTurnsResponse{}
	for i := range list {
		resp.Turns = append(resp.Turns, domainTurnToProto(&list[i]))
	}
	return resp, nil
}

func domainTurnToProto(t *scene.SceneTurn) *pb.SceneTurn {
	actorIDs := make([]*pb.UUID, len(t.ActorIDs))
	for i, a := range t.ActorIDs {
		actorIDs[i] = uuidToProto(a)
	}
	return &pb.SceneTurn{
		Id:         uuidToProto(t.ID),
		NodeId:     uuidToProto(t.NodeID),
		TurnNumber: int32(t.TurnNumber),
		ActorIds:   actorIDs,
		Prompt:     t.Prompt,
		Output:     t.Output,
		Model:      t.Model,
		Status:     t.Status,
		CreatedAt:  timeToProto(t.CreatedAt),
	}
}

type summarySrv struct {
	pb.UnimplementedSummaryServiceServer
	svc compiler.SummaryService
}

func (s *summarySrv) GetSceneSummary(ctx context.Context, req *pb.GetSceneSummaryRequest) (*pb.StorySummary, error) {
	storyID, err := parseProtoUUID(req.StoryId)
	if err != nil {
		return nil, err
	}
	nodeID, err := parseProtoUUID(req.NodeId)
	if err != nil {
		return nil, err
	}
	r, err := s.svc.GetSceneSummary(ctx, storyID, nodeID)
	if err != nil {
		return nil, err
	}
	return domainSummaryToProto(r), nil
}

func (s *summarySrv) GetSummaryByLevel(ctx context.Context, req *pb.GetSummaryByLevelRequest) (*pb.StorySummary, error) {
	storyID, err := parseProtoUUID(req.StoryId)
	if err != nil {
		return nil, err
	}
	r, err := s.svc.GetSummaryByLevel(ctx, storyID, compiler.SummaryLevel(req.Level))
	if err != nil {
		return nil, err
	}
	return domainSummaryToProto(r), nil
}

func (s *summarySrv) CountSummariesByLevel(ctx context.Context, req *pb.CountSummariesByLevelRequest) (*pb.CountSummariesByLevelResponse, error) {
	storyID, err := parseProtoUUID(req.StoryId)
	if err != nil {
		return nil, err
	}
	count, err := s.svc.CountSummariesByLevel(ctx, storyID, compiler.SummaryLevel(req.Level))
	if err != nil {
		return nil, err
	}
	return &pb.CountSummariesByLevelResponse{Count: int32(count)}, nil
}

func (s *summarySrv) ShouldElevate(ctx context.Context, req *pb.ShouldElevateRequest) (*pb.ShouldElevateResponse, error) {
	storyID, err := parseProtoUUID(req.StoryId)
	if err != nil {
		return nil, err
	}
	should, err := s.svc.ShouldElevate(ctx, storyID, compiler.SummaryLevel(req.Level), int(req.Threshold))
	if err != nil {
		return nil, err
	}
	return &pb.ShouldElevateResponse{
		ShouldElevate: should,
		Level:         req.Level,
		Threshold:     req.Threshold,
	}, nil
}

func domainSummaryToProto(s *compiler.StorySummary) *pb.StorySummary {
	p := &pb.StorySummary{
		Id:        uuidToProto(s.ID),
		StoryId:   uuidToProto(s.StoryID),
		Level:     string(s.Level),
		Content:   s.Content,
		WordCount: int32(s.WordCount),
	}
	if s.NodeID != nil {
		p.NodeId = uuidToProto(*s.NodeID)
	}
	return p
}

type storyGenSrv struct {
	pb.UnimplementedStoryGeneratorServiceServer
	svc StoryGenService
}

func (s *storyGenSrv) GenerateStory(ctx context.Context, req *pb.GenerateStoryRequest) (*pb.GenerateStoryResponse, error) {
	storyID, status, err := s.svc.GenerateStory(ctx, req.Synopsis)
	if err != nil {
		return nil, err
	}
	return &pb.GenerateStoryResponse{
		StoryId: storyID,
		Status:  status,
	}, nil
}

// ── Helpers ────────────────────────────────────────────────────

func parseProtoUUID(p *pb.UUID) (uuid.UUID, error) {
	if p == nil {
		return uuid.Nil, status.Error(codes.InvalidArgument, "uuid is nil")
	}
	return uuid.Parse(p.Value)
}

func uuidToProto(id uuid.UUID) *pb.UUID {
	return &pb.UUID{Value: id.String()}
}

func timeToProto(t time.Time) *pb.Timestamp {
	return &pb.Timestamp{Rfc3339: t.Format(time.RFC3339)}
}

// ── Server ─────────────────────────────────────────────────────

type Server struct {
	grpcSrv *grpc.Server
	port    string
}

func New(
	charSvc canonsvc.CharacterService,
	actorSvc canonsvc.ActorService,
	traitSvc canonsvc.TraitService,
	castingSvc canonsvc.CastingService,
	locSvc canonsvc.LocationService,
	loreSvc canonsvc.LoreService,
	storySvc storysvc.StoryService,
	nodeSvc node.Service,
	edgeSvc edge.Service,
	genSvc generation.GenerationService,
	sceneSvc scene.SceneService,
	summarySvc compiler.SummaryService,
	storyGenSvc StoryGenService,
	port string,
) *Server {
	gs := grpc.NewServer()

	pb.RegisterCharacterServiceServer(gs, &characterSrv{svc: charSvc})
	pb.RegisterActorServiceServer(gs, &actorSrv{svc: actorSvc})
	pb.RegisterCharacterTraitServiceServer(gs, &traitSrv{svc: traitSvc})
	pb.RegisterCastingServiceServer(gs, &castingSrv{svc: castingSvc})
	pb.RegisterLocationServiceServer(gs, &locationSrv{svc: locSvc})
	pb.RegisterLoreServiceServer(gs, &loreSrv{svc: loreSvc})
	pb.RegisterStoryServiceServer(gs, &storySrv{storySvc: storySvc, edgeSvc: edgeSvc, nodeSvc: nodeSvc})
	pb.RegisterNodeServiceServer(gs, &nodeSrv{svc: nodeSvc})
	pb.RegisterGenerationServiceServer(gs, &generationSrv{svc: genSvc})
	pb.RegisterSceneServiceServer(gs, &sceneSrv{svc: sceneSvc})
	pb.RegisterSummaryServiceServer(gs, &summarySrv{svc: summarySvc})
	pb.RegisterStoryGeneratorServiceServer(gs, &storyGenSrv{svc: storyGenSvc})

	reflection.Register(gs)

	return &Server{grpcSrv: gs, port: port}
}

func (s *Server) Start(ctx context.Context) error {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", s.port))
	if err != nil {
		return fmt.Errorf("grpc listen: %w", err)
	}
	go func() {
		<-ctx.Done()
		s.grpcSrv.GracefulStop()
	}()
	return s.grpcSrv.Serve(lis)
}

func (s *Server) Stop() {
	s.grpcSrv.GracefulStop()
}
