package vector

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"agent-memory/internal/config"
	"agent-memory/internal/memory/types"

	pb "github.com/qdrant/go-client/qdrant"
)

type qdrantProvider struct {
	conn        *grpc.ClientConn
	points      pb.PointsClient
	collections pb.CollectionsClient
	config      *config.QdrantConfig
}

func newQdrantProvider(cfg *Config) *qdrantProvider {
	qdrantCfg := &config.QdrantConfig{
		URL:        cfg.Qdrant.URL,
		APIKey:     cfg.Qdrant.APIKey,
		Collection: cfg.Qdrant.Collection,
		VectorSize: cfg.VectorSize,
	}

	conn, err := grpc.NewClient(
		qdrantCfg.URL,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		panic(fmt.Sprintf("qdrant dial: %v", err))
	}

	p := &qdrantProvider{
		conn:        conn,
		points:      pb.NewPointsClient(conn),
		collections: pb.NewCollectionsClient(conn),
		config:      qdrantCfg,
	}

	p.ensureCollection(context.Background())

	return p
}

func (p *qdrantProvider) Name() ProviderType { return ProviderQdrant }

func (p *qdrantProvider) ensureCollection(ctx context.Context) {
	collectionName := p.config.Collection
	if collectionName == "" {
		collectionName = "agent_memory"
	}

	_, _ = p.collections.Create(ctx, &pb.CreateCollection{
		CollectionName: collectionName,
		VectorsConfig: &pb.VectorsConfig{
			Config: &pb.VectorsConfig_Params{
				Params: &pb.VectorParams{
					Size:     uint64(p.config.VectorSize),
					Distance: pb.Distance_Cosine,
				},
			},
		},
	})

	_, _ = p.points.CreateFieldIndex(ctx, &pb.CreateFieldIndexCollection{
		CollectionName: collectionName,
		FieldName:      "entity_id",
		FieldType:      pb.FieldType_FieldTypeKeyword.Enum(),
	})
	_, _ = p.points.CreateFieldIndex(ctx, &pb.CreateFieldIndexCollection{
		CollectionName: collectionName,
		FieldName:      "entity_type",
		FieldType:      pb.FieldType_FieldTypeKeyword.Enum(),
	})
	_, _ = p.points.CreateFieldIndex(ctx, &pb.CreateFieldIndexCollection{
		CollectionName: collectionName,
		FieldName:      "user_id",
		FieldType:      pb.FieldType_FieldTypeKeyword.Enum(),
	})
}

func (p *qdrantProvider) StoreEmbedding(ctx context.Context, text string, id string, embedding []float32, meta map[string]interface{}) (string, error) {
	collectionName := p.config.Collection
	if collectionName == "" {
		collectionName = "agent_memory"
	}

	payload := map[string]*pb.Value{
		"text":          {Kind: &pb.Value_StringValue{StringValue: text}},
		"entity_id":     {Kind: &pb.Value_StringValue{StringValue: id}},
		"created_at":    {Kind: &pb.Value_StringValue{StringValue: time.Now().Format(time.RFC3339)}},
		"last_accessed": {Kind: &pb.Value_StringValue{StringValue: time.Now().Format(time.RFC3339)}},
	}

	for k, v := range meta {
		payload[k] = toValue(v)
	}

	_, err := p.points.Upsert(ctx, &pb.UpsertPoints{
		CollectionName: collectionName,
		Points: []*pb.PointStruct{
			{
				Id:      &pb.PointId{PointIdOptions: &pb.PointId_Uuid{Uuid: id}},
				Vectors: &pb.Vectors{VectorsOptions: &pb.Vectors_Vector{Vector: &pb.Vector{Data: embedding}}},
				Payload: payload,
			},
		},
	})
	if err != nil {
		return "", fmt.Errorf("store embedding: %w", err)
	}

	return id, nil
}

func (p *qdrantProvider) Search(ctx context.Context, query []float32, limit int, threshold float32, filters map[string]interface{}) ([]types.MemoryResult, error) {
	collectionName := p.config.Collection
	if collectionName == "" {
		collectionName = "agent_memory"
	}

	var filter *pb.Filter
	if len(filters) > 0 {
		filter = buildFilter(filters)
	}

	result, err := p.points.Search(ctx, &pb.SearchPoints{
		CollectionName: collectionName,
		Vector:         query,
		Limit:          uint64(limit),
		ScoreThreshold: &threshold,
		Filter:         filter,
		WithPayload:    &pb.WithPayloadSelector{SelectorOptions: &pb.WithPayloadSelector_Enable{Enable: true}},
	})
	if err != nil {
		return nil, fmt.Errorf("search: %w", err)
	}

	var results []types.MemoryResult
	for _, hit := range result.Result {
		payload := map[string]interface{}{}
		for k, v := range hit.Payload {
			payload[k] = fromValue(v)
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
			Entity: types.Entity{ID: entityID, Properties: payload},
			Score:  hit.Score,
			Text:   text,
			Source: "qdrant",
		})
	}

	return results, nil
}

func (p *qdrantProvider) UpdateMemory(ctx context.Context, id string, text string, meta map[string]interface{}) error {
	collectionName := p.config.Collection
	if collectionName == "" {
		collectionName = "agent_memory"
	}

	payload := map[string]*pb.Value{
		"text":          {Kind: &pb.Value_StringValue{StringValue: text}},
		"last_accessed": {Kind: &pb.Value_StringValue{StringValue: time.Now().Format(time.RFC3339)}},
	}
	for k, v := range meta {
		payload[k] = toValue(v)
	}

	_, err := p.points.SetPayload(ctx, &pb.SetPayloadPoints{
		CollectionName: collectionName,
		PointsSelector: &pb.PointsSelector{
			PointsSelectorOneOf: &pb.PointsSelector_Points{
				Points: &pb.PointsIdsList{
					Ids: []*pb.PointId{{PointIdOptions: &pb.PointId_Uuid{Uuid: id}}},
				},
			},
		},
		Payload: payload,
	})
	return err
}

func (p *qdrantProvider) DeleteMemory(ctx context.Context, id string) error {
	collectionName := p.config.Collection
	if collectionName == "" {
		collectionName = "agent_memory"
	}

	_, err := p.points.Delete(ctx, &pb.DeletePoints{
		CollectionName: collectionName,
		Points: &pb.PointsSelector{
			PointsSelectorOneOf: &pb.PointsSelector_Points{
				Points: &pb.PointsIdsList{
					Ids: []*pb.PointId{{PointIdOptions: &pb.PointId_Uuid{Uuid: id}}},
				},
			},
		},
	})
	return err
}

func (p *qdrantProvider) UpdateVector(ctx context.Context, id string, embedding []float32) error {
	collectionName := p.config.Collection
	if collectionName == "" {
		collectionName = "agent_memory"
	}

	_, err := p.points.Upsert(ctx, &pb.UpsertPoints{
		CollectionName: collectionName,
		Points: []*pb.PointStruct{
			{
				Id:      &pb.PointId{PointIdOptions: &pb.PointId_Uuid{Uuid: id}},
				Vectors: &pb.Vectors{VectorsOptions: &pb.Vectors_Vector{Vector: &pb.Vector{Data: embedding}}},
			},
		},
	})
	return err
}

func (p *qdrantProvider) DeleteByFilter(ctx context.Context, filter map[string]interface{}) (int, error) {
	return 0, nil
}

func (p *qdrantProvider) Ping(ctx context.Context) error {
	return nil
}

func (p *qdrantProvider) Close() error {
	return p.conn.Close()
}

func toValue(v interface{}) *pb.Value {
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

func fromValue(v *pb.Value) interface{} {
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
