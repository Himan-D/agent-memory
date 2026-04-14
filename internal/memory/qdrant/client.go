package qdrant

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"

	"agent-memory/internal/config"
	"agent-memory/internal/memory/types"

	pb "github.com/qdrant/go-client/qdrant"
)

const (
	CollectionName = "agent_long_term_memory"
	VectorSize     = 1536
)

type Client struct {
	conn       *grpc.ClientConn
	collection pb.CollectionsClient
	points     pb.PointsClient
	config     config.QdrantConfig
}

func NewClient(cfg config.QdrantConfig) (*Client, error) {
	conn, err := grpc.NewClient(
		cfg.URL,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, fmt.Errorf("qdrant dial: %w", err)
	}

	c := &Client{
		conn:       conn,
		collection: pb.NewCollectionsClient(conn),
		points:     pb.NewPointsClient(conn),
		config:     cfg,
	}

	// Ensure collection exists
	if err := c.ensureCollection(context.Background()); err != nil {
		return nil, fmt.Errorf("ensure collection: %w", err)
	}

	return c, nil
}

func (c *Client) Close() error {
	return c.conn.Close()
}

func (c *Client) Ping(ctx context.Context) error {
	_, err := c.collection.Get(ctx, &pb.GetCollectionInfoRequest{
		CollectionName: CollectionName,
	})
	return err
}

func (c *Client) ensureCollection(ctx context.Context) error {
	// Check if collection exists
	_, err := c.collection.Get(ctx, &pb.GetCollectionInfoRequest{
		CollectionName: CollectionName,
	})
	if err == nil {
		return nil // Collection already exists
	}

	// Create collection with proper pointer types
	m := uint64(16)
	efConstruct := uint64(200)
	fullScanThreshold := uint64(10000)
	memmapThreshold := uint64(20000)

	_, err = c.collection.Create(ctx, &pb.CreateCollection{
		CollectionName: CollectionName,
		VectorsConfig: &pb.VectorsConfig{
			Config: &pb.VectorsConfig_Params{
				Params: &pb.VectorParams{
					Size:     uint64(VectorSize),
					Distance: pb.Distance_Cosine,
					HnswConfig: &pb.HnswConfigDiff{
						M:                 &m,
						EfConstruct:       &efConstruct,
						FullScanThreshold: &fullScanThreshold,
					},
				},
			},
		},
		OptimizersConfig: &pb.OptimizersConfigDiff{
			MemmapThreshold: &memmapThreshold,
		},
	})
	if err != nil {
		return fmt.Errorf("create collection: %w", err)
	}

	// Create payload index for entity_id
	_, err = c.points.CreateFieldIndex(ctx, &pb.CreateFieldIndexCollection{
		CollectionName: CollectionName,
		FieldName:      "entity_id",
		FieldType:      pb.FieldType_FieldTypeKeyword.Enum(),
	})
	if err != nil {
		return fmt.Errorf("create entity_id index: %w", err)
	}

	// Create payload index for entity_type
	_, err = c.points.CreateFieldIndex(ctx, &pb.CreateFieldIndexCollection{
		CollectionName: CollectionName,
		FieldName:      "entity_type",
		FieldType:      pb.FieldType_FieldTypeKeyword.Enum(),
	})
	if err != nil {
		return fmt.Errorf("create entity_type index: %w", err)
	}

	return nil
}

func (c *Client) StoreEmbedding(
	ctx context.Context,
	text string,
	entityID string,
	embedding []float32,
	meta map[string]interface{},
) (string, error) {
	pointID := uuid.New().String()

	payload := map[string]*pb.Value{
		"text":          {Kind: &pb.Value_StringValue{StringValue: text}},
		"entity_id":     {Kind: &pb.Value_StringValue{StringValue: entityID}},
		"created_at":    {Kind: &pb.Value_StringValue{StringValue: time.Now().Format(time.RFC3339)}},
		"last_accessed": {Kind: &pb.Value_StringValue{StringValue: time.Now().Format(time.RFC3339)}},
	}

	// Add metadata
	for k, v := range meta {
		payload[k] = toQdrantValue(v)
	}

	_, err := c.points.Upsert(ctx, &pb.UpsertPoints{
		CollectionName: CollectionName,
		Points: []*pb.PointStruct{
			{
				Id:      &pb.PointId{PointIdOptions: &pb.PointId_Uuid{Uuid: pointID}},
				Vectors: &pb.Vectors{VectorsOptions: &pb.Vectors_Vector{Vector: &pb.Vector{Data: embedding}}},
				Payload: payload,
			},
		},
	})
	if err != nil {
		return "", fmt.Errorf("store embedding: %w", err)
	}

	return pointID, nil
}

func (c *Client) SearchSemantic(
	ctx context.Context,
	query []float32,
	limit int,
	scoreThreshold float32,
	filters map[string]interface{},
) ([]types.MemoryResult, error) {
	var filter *pb.Filter
	if len(filters) > 0 {
		filter = buildFilter(filters)
	}

	result, err := c.points.Search(ctx, &pb.SearchPoints{
		CollectionName: CollectionName,
		Vector:         query,
		Limit:          uint64(limit),
		ScoreThreshold: &scoreThreshold,
		Filter:         filter,
		WithPayload:    &pb.WithPayloadSelector{SelectorOptions: &pb.WithPayloadSelector_Enable{Enable: true}},
	})
	if err != nil {
		return nil, fmt.Errorf("search semantic: %w", err)
	}

	var results []types.MemoryResult
	for _, hit := range result.Result {
		payload := map[string]interface{}{}
		for k, v := range hit.Payload {
			payload[k] = fromQdrantValue(v)
		}

		text := ""
		if t, ok := payload["text"].(string); ok {
			text = t
		}
		entityID := ""
		if eid, ok := payload["entity_id"].(string); ok {
			entityID = eid
		}

		results = append(results, types.MemoryResult{
			Entity: types.Entity{
				ID:         entityID,
				Properties: payload,
			},
			Score:  hit.Score,
			Text:   text,
			Source: "qdrant",
		})
	}

	return results, nil
}

func (c *Client) Search(ctx context.Context, query []float32, limit int, threshold float32, filters map[string]interface{}) ([]types.MemoryResult, error) {
	return c.SearchSemantic(ctx, query, limit, threshold, filters)
}

func (c *Client) UpdateMemory(
	ctx context.Context,
	id string,
	text string,
	meta map[string]interface{},
) error {
	payload := map[string]*pb.Value{
		"text":          {Kind: &pb.Value_StringValue{StringValue: text}},
		"last_accessed": {Kind: &pb.Value_StringValue{StringValue: time.Now().Format(time.RFC3339)}},
	}
	for k, v := range meta {
		payload[k] = toQdrantValue(v)
	}

	_, err := c.points.SetPayload(ctx, &pb.SetPayloadPoints{
		CollectionName: CollectionName,
		PointsSelector: &pb.PointsSelector{
			PointsSelectorOneOf: &pb.PointsSelector_Points{
				Points: &pb.PointsIdsList{
					Ids: []*pb.PointId{{PointIdOptions: &pb.PointId_Uuid{Uuid: id}}},
				},
			},
		},
		Payload: payload,
	})
	if err != nil {
		return fmt.Errorf("update memory: %w", err)
	}
	return nil
}

func (c *Client) DeleteMemory(ctx context.Context, id string) error {
	_, err := c.points.Delete(ctx, &pb.DeletePoints{
		CollectionName: CollectionName,
		Points: &pb.PointsSelector{
			PointsSelectorOneOf: &pb.PointsSelector_Points{
				Points: &pb.PointsIdsList{
					Ids: []*pb.PointId{{PointIdOptions: &pb.PointId_Uuid{Uuid: id}}},
				},
			},
		},
	})
	if err != nil {
		return fmt.Errorf("delete memory: %w", err)
	}
	return nil
}

func (c *Client) DeleteByFilter(ctx context.Context, filter map[string]interface{}) (int, error) {
	pbFilter := buildFilter(filter)

	result, err := c.points.Delete(ctx, &pb.DeletePoints{
		CollectionName: CollectionName,
		Points: &pb.PointsSelector{
			PointsSelectorOneOf: &pb.PointsSelector_Filter{
				Filter: pbFilter,
			},
		},
	})
	if err != nil {
		return 0, fmt.Errorf("delete by filter: %w", err)
	}

	if result.Result != nil && result.Result.Status == pb.UpdateStatus_Completed {
		return 1, nil
	}

	return 0, nil
}

func (c *Client) GetByEntityID(ctx context.Context, entityID string) ([]types.MemoryResult, error) {
	filter := &pb.Filter{
		Must: []*pb.Condition{
			{
				ConditionOneOf: &pb.Condition_Field{
					Field: &pb.FieldCondition{
						Key: "entity_id",
						Match: &pb.Match{
							MatchValue: &pb.Match_Keyword{Keyword: entityID},
						},
					},
				},
			},
		},
	}

	result, err := c.points.Scroll(ctx, &pb.ScrollPoints{
		CollectionName: CollectionName,
		Filter:         filter,
		WithPayload:    &pb.WithPayloadSelector{SelectorOptions: &pb.WithPayloadSelector_Enable{Enable: true}},
	})
	if err != nil {
		return nil, fmt.Errorf("get by entity id: %w", err)
	}

	var results []types.MemoryResult
	for _, point := range result.Result {
		payload := map[string]interface{}{}
		for k, v := range point.Payload {
			payload[k] = fromQdrantValue(v)
		}

		results = append(results, types.MemoryResult{
			Entity: types.Entity{
				ID:         entityID,
				Properties: payload,
			},
			Source: "qdrant",
		})
	}
	return results, nil
}

func (c *Client) WithAPIKey(ctx context.Context) context.Context {
	if c.config.APIKey != "" {
		ctx = metadata.AppendToOutgoingContext(ctx, "api-key", c.config.APIKey)
	}
	return ctx
}

func toQdrantValue(v interface{}) *pb.Value {
	switch val := v.(type) {
	case string:
		return &pb.Value{Kind: &pb.Value_StringValue{StringValue: val}}
	case int:
		return &pb.Value{Kind: &pb.Value_IntegerValue{IntegerValue: int64(val)}}
	case int64:
		return &pb.Value{Kind: &pb.Value_IntegerValue{IntegerValue: val}}
	case float64:
		return &pb.Value{Kind: &pb.Value_DoubleValue{DoubleValue: val}}
	case float32:
		return &pb.Value{Kind: &pb.Value_DoubleValue{DoubleValue: float64(val)}}
	case bool:
		return &pb.Value{Kind: &pb.Value_BoolValue{BoolValue: val}}
	default:
		return &pb.Value{Kind: &pb.Value_StringValue{StringValue: fmt.Sprintf("%v", val)}}
	}
}

func fromQdrantValue(v *pb.Value) interface{} {
	switch val := v.Kind.(type) {
	case *pb.Value_StringValue:
		return val.StringValue
	case *pb.Value_IntegerValue:
		return val.IntegerValue
	case *pb.Value_DoubleValue:
		return val.DoubleValue
	case *pb.Value_BoolValue:
		return val.BoolValue
	default:
		return nil
	}
}

func buildFilter(filters map[string]interface{}) *pb.Filter {
	var conditions []*pb.Condition
	for k, v := range filters {
		conditions = append(conditions, &pb.Condition{
			ConditionOneOf: &pb.Condition_Field{
				Field: &pb.FieldCondition{
					Key: k,
					Match: &pb.Match{
						MatchValue: &pb.Match_Keyword{Keyword: fmt.Sprintf("%v", v)},
					},
				},
			},
		})
	}
	return &pb.Filter{Must: conditions}
}

func (c *Client) UpdateVector(ctx context.Context, id string, embedding []float32) error {
	_, err := c.points.Upsert(ctx, &pb.UpsertPoints{
		CollectionName: CollectionName,
		Points: []*pb.PointStruct{
			{
				Id:      &pb.PointId{PointIdOptions: &pb.PointId_Uuid{Uuid: id}},
				Vectors: &pb.Vectors{VectorsOptions: &pb.Vectors_Vector{Vector: &pb.Vector{Data: embedding}}},
			},
		},
	})
	return err
}
