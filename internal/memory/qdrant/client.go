package qdrant

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"

	"agent-memory/internal/config"
	"agent-memory/internal/memory/types"
	"agent-memory/internal/tenant"

	pb "github.com/qdrant/go-client/qdrant"
)

const (
	CollectionName = "agent_long_term_memory"
)

type Client struct {
	conn              *grpc.ClientConn
	collection        pb.CollectionsClient
	points            pb.PointsClient
	config            config.QdrantConfig
	perTenant         bool
	collectionPrefix  string
	ensuredCollections sync.Map // collection name -> struct{}
}

func NewClient(cfg config.QdrantConfig) (*Client, error) {
	return NewClientWithTenant(cfg, true, cfg.Collection)
}

// NewClientWithTenant creates a Qdrant client with optional per-tenant collections.
// When perTenant is true, collection names are {prefix}_{tenant_slug} from request context.
func NewClientWithTenant(cfg config.QdrantConfig, perTenant bool, prefix string) (*Client, error) {
	// Convert HTTP URL to gRPC URL format
	grpcURL := cfg.URL
	if strings.HasPrefix(grpcURL, "http://") {
		grpcURL = grpcURL[7:] // Remove http:// prefix
	} else if strings.HasPrefix(grpcURL, "https://") {
		grpcURL = grpcURL[8:] // Remove https:// prefix
	}

	conn, err := grpc.NewClient(
		grpcURL,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(100*1024*1024)),
	)
	if err != nil {
		return nil, fmt.Errorf("qdrant dial: %w", err)
	}

	if prefix == "" {
		prefix = cfg.Collection
	}
	if prefix == "" {
		prefix = "agent_memory"
	}

	c := &Client{
		conn:             conn,
		collection:       pb.NewCollectionsClient(conn),
		points:           pb.NewPointsClient(conn),
		config:           cfg,
		perTenant:        perTenant,
		collectionPrefix: prefix,
	}

	// Ensure base/default collection exists for health checks and legacy single-tenant mode.
	if err := c.ensureNamedCollection(context.Background(), c.baseCollectionName()); err != nil {
		return nil, fmt.Errorf("ensure collection: %w", err)
	}

	return c, nil
}

func (c *Client) Close() error {
	return c.conn.Close()
}

func (c *Client) baseCollectionName() string {
	if c.config.Collection != "" {
		return c.config.Collection
	}
	return CollectionName
}

// collectionName resolves the collection for this request.
// When per-tenant mode is on and ctx carries a tenant ID, uses agent_memory_{tenant}.
func (c *Client) collectionName(ctx context.Context) string {
	if c.perTenant {
		if tid := tenant.IDFromContext(ctx); tid != "" {
			return tenant.CollectionName(c.collectionPrefix, tid)
		}
		// Fallback: tenant_id in a reserved context value via filters is handled by callers;
		// without tenant, use base collection (legacy / admin bulk).
	}
	return c.baseCollectionName()
}

// CollectionForTenant returns the Qdrant collection name for a tenant (no ensure).
func (c *Client) CollectionForTenant(tenantID string) string {
	if !c.perTenant || tenantID == "" {
		return c.baseCollectionName()
	}
	return tenant.CollectionName(c.collectionPrefix, tenantID)
}

func (c *Client) Ping(ctx context.Context) error {
	_, err := c.collection.Get(ctx, &pb.GetCollectionInfoRequest{
		CollectionName: c.collectionName(ctx),
	})
	return err
}

func (c *Client) ensureCollection(ctx context.Context) error {
	return c.ensureNamedCollection(ctx, c.collectionName(ctx))
}

func (c *Client) ensureNamedCollection(ctx context.Context, name string) error {
	if name == "" {
		name = c.baseCollectionName()
	}
	if _, ok := c.ensuredCollections.Load(name); ok {
		return nil
	}

	// Check if collection exists
	_, err := c.collection.Get(ctx, &pb.GetCollectionInfoRequest{
		CollectionName: name,
	})
	if err == nil {
		c.ensuredCollections.Store(name, struct{}{})
		return nil
	}

	// Create collection with proper pointer types
	m := uint64(16)
	efConstruct := uint64(200)
	fullScanThreshold := uint64(10000)
	memmapThreshold := uint64(20000)

	_, err = c.collection.Create(ctx, &pb.CreateCollection{
		CollectionName: name,
		VectorsConfig: &pb.VectorsConfig{
			Config: &pb.VectorsConfig_Params{
				Params: &pb.VectorParams{
					Size:     uint64(c.config.VectorSize),
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
		// Another replica may have created it; re-check.
		if _, getErr := c.collection.Get(ctx, &pb.GetCollectionInfoRequest{CollectionName: name}); getErr == nil {
			c.ensuredCollections.Store(name, struct{}{})
			return nil
		}
		return fmt.Errorf("create collection %s: %w", name, err)
	}

	// Create payload indexes for common filters
	for _, field := range []string{"entity_id", "entity_type", "tenant_id", "user_id", "org_id", "memory_id"} {
		_, idxErr := c.points.CreateFieldIndex(ctx, &pb.CreateFieldIndexCollection{
			CollectionName: name,
			FieldName:      field,
			FieldType:      pb.FieldType_FieldTypeKeyword.Enum(),
		})
		if idxErr != nil {
			// Non-fatal: index may already exist
			_ = idxErr
		}
	}

	c.ensuredCollections.Store(name, struct{}{})
	return nil
}

func (c *Client) StoreEmbedding(
	ctx context.Context,
	text string,
	id string,
	embedding []float32,
	meta map[string]interface{},
) (string, error) {
	pointID := qdrantPointID(id)
	coll := c.collectionName(ctx)
	if err := c.ensureNamedCollection(ctx, coll); err != nil {
		return "", err
	}

	payload := map[string]*pb.Value{
		"text":          {Kind: &pb.Value_StringValue{StringValue: text}},
		"memory_id":     {Kind: &pb.Value_StringValue{StringValue: id}},
		"created_at":    {Kind: &pb.Value_StringValue{StringValue: time.Now().Format(time.RFC3339)}},
		"last_accessed": {Kind: &pb.Value_StringValue{StringValue: time.Now().Format(time.RFC3339)}},
	}

	// Add metadata
	for k, v := range meta {
		payload[k] = toQdrantValue(v)
	}
	// Always stamp tenant_id when present on context (defense in depth).
	if tid := tenant.IDFromContext(ctx); tid != "" {
		if _, ok := payload["tenant_id"]; !ok {
			payload["tenant_id"] = &pb.Value{Kind: &pb.Value_StringValue{StringValue: tid}}
		}
	}

	_, err := c.points.Upsert(ctx, &pb.UpsertPoints{
		CollectionName: coll,
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
	coll := c.collectionName(ctx)
	if err := c.ensureNamedCollection(ctx, coll); err != nil {
		return nil, err
	}

	// Defense in depth: always filter by tenant when present on context.
	if tid := tenant.IDFromContext(ctx); tid != "" {
		if filters == nil {
			filters = map[string]interface{}{}
		}
		if _, ok := filters["tenant_id"]; !ok {
			filters["tenant_id"] = tid
		}
	}

	var filter *pb.Filter
	if len(filters) > 0 {
		filter = buildFilter(filters)
	}

	result, err := c.points.Search(ctx, &pb.SearchPoints{
		CollectionName: coll,
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
		memoryID := ""
		if mid, ok := payload["memory_id"].(string); ok && mid != "" {
			memoryID = mid
		}

		results = append(results, types.MemoryResult{
			Entity: types.Entity{
				ID:         entityID,
				Properties: payload,
			},
			Score:    hit.Score,
			Text:     text,
			Source:   "qdrant",
			MemoryID: memoryID,
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
	coll := c.collectionName(ctx)
	if err := c.ensureNamedCollection(ctx, coll); err != nil {
		return err
	}
	payload := map[string]*pb.Value{
		"text":          {Kind: &pb.Value_StringValue{StringValue: text}},
		"last_accessed": {Kind: &pb.Value_StringValue{StringValue: time.Now().Format(time.RFC3339)}},
	}
	for k, v := range meta {
		payload[k] = toQdrantValue(v)
	}

	_, err := c.points.SetPayload(ctx, &pb.SetPayloadPoints{
		CollectionName: coll,
		PointsSelector: &pb.PointsSelector{
			PointsSelectorOneOf: &pb.PointsSelector_Points{
				Points: &pb.PointsIdsList{Ids: []*pb.PointId{{PointIdOptions: &pb.PointId_Uuid{Uuid: qdrantPointID(id)}}}},
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
	coll := c.collectionName(ctx)
	_, err := c.points.Delete(ctx, &pb.DeletePoints{
		CollectionName: coll,
		Points: &pb.PointsSelector{
			PointsSelectorOneOf: &pb.PointsSelector_Points{
				Points: &pb.PointsIdsList{Ids: []*pb.PointId{{PointIdOptions: &pb.PointId_Uuid{Uuid: qdrantPointID(id)}}}},
			},
		},
	})
	if err != nil {
		return fmt.Errorf("delete memory: %w", err)
	}
	return nil
}

func (c *Client) DeleteByFilter(ctx context.Context, filter map[string]interface{}) (int, error) {
	coll := c.collectionName(ctx)
	pbFilter := buildFilter(filter)

	result, err := c.points.Delete(ctx, &pb.DeletePoints{
		CollectionName: coll,
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
		CollectionName: c.collectionName(ctx),
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
		text := ""
		if t, ok := payload["text"].(string); ok {
			text = t
		}
		memoryID := entityID
		if mid, ok := payload["memory_id"].(string); ok && mid != "" {
			memoryID = mid
		}

		results = append(results, types.MemoryResult{
			Entity: types.Entity{
				ID:         entityID,
				Properties: payload,
			},
			Score:    1.0,
			Text:     text,
			Source:   "qdrant",
			MemoryID: memoryID,
		})
	}
	return results, nil
}

func qdrantPointID(id string) string {
	if id == "" {
		return uuid.New().String()
	}
	if parsed, err := uuid.Parse(id); err == nil {
		return parsed.String()
	}
	return uuid.NewSHA1(uuid.NameSpaceOID, []byte(id)).String()
}

func (c *Client) WithAPIKey(ctx context.Context) context.Context {
	if c.config.APIKey != "" {
		ctx = metadata.AppendToOutgoingContext(ctx, "api-key", c.config.APIKey)
	}
	return ctx
}

func toQdrantValue(v interface{}) *pb.Value {
	if converted, err := pb.NewValue(normalizeQdrantValue(v)); err == nil {
		return converted
	}
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

func normalizeQdrantValue(v interface{}) interface{} {
	switch val := v.(type) {
	case []string:
		out := make([]interface{}, 0, len(val))
		for _, item := range val {
			out = append(out, item)
		}
		return out
	case []int:
		out := make([]interface{}, 0, len(val))
		for _, item := range val {
			out = append(out, item)
		}
		return out
	case []float64:
		out := make([]interface{}, 0, len(val))
		for _, item := range val {
			out = append(out, item)
		}
		return out
	case []bool:
		out := make([]interface{}, 0, len(val))
		for _, item := range val {
			out = append(out, item)
		}
		return out
	default:
		return v
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
	case *pb.Value_ListValue:
		values := val.ListValue.GetValues()
		out := make([]interface{}, 0, len(values))
		for _, item := range values {
			out = append(out, fromQdrantValue(item))
		}
		return out
	default:
		return nil
	}
}

// buildFilter converts a flat string→value filter map to a Qdrant Filter.
// The special key "__op_filters__" may hold a []types.SearchFilter slice with
// richer operator semantics (gt, lt, gte, lte, in, not_eq, contains).
func buildFilter(filters map[string]interface{}) *pb.Filter {
	var mustConditions []*pb.Condition
	var mustNotConditions []*pb.Condition

	// Handle rich operator filters injected by filtersToMap.
	if raw, ok := filters["__op_filters__"]; ok {
		if rules, ok := raw.([]types.SearchFilter); ok {
			must, mustNot := buildOperatorConditions(rules)
			mustConditions = append(mustConditions, must...)
			mustNotConditions = append(mustNotConditions, mustNot...)
		}
	}

	for k, v := range filters {
		if k == "__op_filters__" {
			continue
		}
		mustConditions = append(mustConditions, &pb.Condition{
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

	f := &pb.Filter{}
	if len(mustConditions) > 0 {
		f.Must = mustConditions
	}
	if len(mustNotConditions) > 0 {
		f.MustNot = mustNotConditions
	}
	return f
}

// buildOperatorConditions converts a slice of SearchFilter rules with operators
// into (must, mustNot) Qdrant condition slices.
// Supported operators: eq (default), not_eq, gt, lt, gte, lte, in, contains.
func buildOperatorConditions(rules []types.SearchFilter) (must, mustNot []*pb.Condition) {
	for _, rule := range rules {
		op := rule.Operator
		if op == "" {
			op = "eq"
		}
		switch op {
		case "eq":
			must = append(must, &pb.Condition{
				ConditionOneOf: &pb.Condition_Field{
					Field: &pb.FieldCondition{
						Key:   rule.Field,
						Match: &pb.Match{MatchValue: &pb.Match_Keyword{Keyword: fmt.Sprintf("%v", rule.Value)}},
					},
				},
			})
		case "not_eq":
			mustNot = append(mustNot, &pb.Condition{
				ConditionOneOf: &pb.Condition_Field{
					Field: &pb.FieldCondition{
						Key:   rule.Field,
						Match: &pb.Match{MatchValue: &pb.Match_Keyword{Keyword: fmt.Sprintf("%v", rule.Value)}},
					},
				},
			})
		case "gt", "lt", "gte", "lte":
			fval := toFloat64(rule.Value)
			r := &pb.Range{}
			switch op {
			case "gt":
				r.Gt = &fval
			case "lt":
				r.Lt = &fval
			case "gte":
				r.Gte = &fval
			case "lte":
				r.Lte = &fval
			}
			must = append(must, &pb.Condition{
				ConditionOneOf: &pb.Condition_Field{
					Field: &pb.FieldCondition{
						Key:   rule.Field,
						Range: r,
					},
				},
			})
		case "in":
			keywords := toStringSlice(rule.Value)
			if len(keywords) > 0 {
				must = append(must, &pb.Condition{
					ConditionOneOf: &pb.Condition_Field{
						Field: &pb.FieldCondition{
							Key: rule.Field,
							Match: &pb.Match{
								MatchValue: &pb.Match_Keywords{
									Keywords: &pb.RepeatedStrings{Strings: keywords},
								},
							},
						},
					},
				})
			}
		case "contains":
			must = append(must, &pb.Condition{
				ConditionOneOf: &pb.Condition_Field{
					Field: &pb.FieldCondition{
						Key:   rule.Field,
						Match: &pb.Match{MatchValue: &pb.Match_Text{Text: fmt.Sprintf("%v", rule.Value)}},
					},
				},
			})
		}
	}
	return must, mustNot
}

// toFloat64 coerces common numeric types to float64.
func toFloat64(v interface{}) float64 {
	switch n := v.(type) {
	case float64:
		return n
	case float32:
		return float64(n)
	case int:
		return float64(n)
	case int32:
		return float64(n)
	case int64:
		return float64(n)
	default:
		return 0
	}
}

// toStringSlice converts []interface{} or []string to []string.
func toStringSlice(v interface{}) []string {
	switch vals := v.(type) {
	case []string:
		return vals
	case []interface{}:
		out := make([]string, 0, len(vals))
		for _, item := range vals {
			out = append(out, fmt.Sprintf("%v", item))
		}
		return out
	default:
		return nil
	}
}

func (c *Client) UpdateVector(ctx context.Context, id string, embedding []float32) error {
	coll := c.collectionName(ctx)
	_, err := c.points.UpdateVectors(ctx, &pb.UpdatePointVectors{
		CollectionName: coll,
		Points: []*pb.PointVectors{
			{
				Id:      &pb.PointId{PointIdOptions: &pb.PointId_Uuid{Uuid: qdrantPointID(id)}},
				Vectors: &pb.Vectors{VectorsOptions: &pb.Vectors_Vector{Vector: &pb.Vector{Data: embedding}}},
			},
		},
	})
	return err
}

func (c *Client) BatchStoreEmbeddings(ctx context.Context, items []types.BatchEmbeddingItem) error {
	const batchSize = 500
	now := time.Now().Format(time.RFC3339)
	coll := c.collectionName(ctx)
	if err := c.ensureNamedCollection(ctx, coll); err != nil {
		return err
	}
	tid := tenant.IDFromContext(ctx)

	for i := 0; i < len(items); i += batchSize {
		end := i + batchSize
		if end > len(items) {
			end = len(items)
		}
		chunk := items[i:end]

		points := make([]*pb.PointStruct, 0, len(chunk))
		for _, item := range chunk {
			payload := map[string]*pb.Value{
				"text":          {Kind: &pb.Value_StringValue{StringValue: item.Text}},
				"memory_id":     {Kind: &pb.Value_StringValue{StringValue: item.ID}},
				"created_at":    {Kind: &pb.Value_StringValue{StringValue: now}},
				"last_accessed": {Kind: &pb.Value_StringValue{StringValue: now}},
			}
			for k, v := range item.Metadata {
				payload[k] = toQdrantValue(v)
			}
			if tid != "" {
				if _, ok := payload["tenant_id"]; !ok {
					payload["tenant_id"] = &pb.Value{Kind: &pb.Value_StringValue{StringValue: tid}}
				}
			}
			points = append(points, &pb.PointStruct{
				Id:      &pb.PointId{PointIdOptions: &pb.PointId_Uuid{Uuid: qdrantPointID(item.ID)}},
				Vectors: &pb.Vectors{VectorsOptions: &pb.Vectors_Vector{Vector: &pb.Vector{Data: item.Embedding}}},
				Payload: payload,
			})
		}

		wait := true
		if _, err := c.points.Upsert(ctx, &pb.UpsertPoints{
			CollectionName: coll,
			Wait:           &wait,
			Points:         points,
		}); err != nil {
			return fmt.Errorf("batch upsert %d-%d: %w", i, end, err)
		}
	}
	return nil
}
