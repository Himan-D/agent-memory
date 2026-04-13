package datapoint

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type DataPoint struct {
	ID         string                 `json:"id"`
	Type       DataPointType          `json:"type"`
	Content    string                 `json:"content"`
	Summary    string                 `json:"summary,omitempty"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
	Source     SourceInfo             `json:"source"`
	Embedding  []float32              `json:"-"`
	CreatedAt  time.Time              `json:"created_at"`
	UpdatedAt  time.Time              `json:"updated_at"`
	IsPartOf   *PartOfInfo            `json:"is_part_of,omitempty"`
	Properties map[string]string      `json:"properties,omitempty"`
}

type DataPointType string

const (
	DataPointTypeDocument DataPointType = "document"
	DataPointTypeChunk    DataPointType = "chunk"
	DataPointTypeEntity   DataPointType = "entity"
	DataPointTypeRelation DataPointType = "relation"
	DataPointTypeSummary  DataPointType = "summary"
	DataPointTypeConcept  DataPointType = "concept"
)

type SourceInfo struct {
	Type      string `json:"type"`
	Name      string `json:"name"`
	Location  string `json:"location,omitempty"`
	LineStart int    `json:"line_start,omitempty"`
	LineEnd   int    `json:"line_end,omitempty"`
}

type PartOfInfo struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Type     string `json:"type"`
	Location string `json:"location,omitempty"`
}

func New(content string, dpType DataPointType) *DataPoint {
	now := time.Now()
	return &DataPoint{
		ID:        uuid.New().String(),
		Type:      dpType,
		Content:   content,
		Metadata:  make(map[string]interface{}),
		Source:    SourceInfo{Type: "unknown"},
		CreatedAt: now,
		UpdatedAt: now,
	}
}

func NewChunk(content string, parent *DataPoint, chunkIndex int) *DataPoint {
	dp := New(content, DataPointTypeChunk)
	if parent != nil {
		dp.IsPartOf = &PartOfInfo{
			ID:       parent.ID,
			Name:     parent.Source.Name,
			Type:     string(parent.Type),
			Location: parent.Source.Location,
		}
	}
	dp.Metadata["chunk_index"] = chunkIndex
	return dp
}

func NewEntity(name string, entityType string, properties map[string]string) *DataPoint {
	dp := New(name, DataPointTypeEntity)
	dp.Properties = properties
	if dp.Properties == nil {
		dp.Properties = make(map[string]string)
	}
	dp.Properties["entity_type"] = entityType
	return dp
}

func NewRelation(sourceID, targetID, relationType string, properties map[string]string) *DataPoint {
	dp := New(fmt.Sprintf("%s -> %s", sourceID, targetID), DataPointTypeRelation)
	dp.Properties = properties
	if dp.Properties == nil {
		dp.Properties = make(map[string]string)
	}
	dp.Properties["source_id"] = sourceID
	dp.Properties["target_id"] = targetID
	dp.Properties["relation_type"] = relationType
	return dp
}

func (d *DataPoint) SetSource(sourceType, name, location string) {
	d.Source = SourceInfo{
		Type:     sourceType,
		Name:     name,
		Location: location,
	}
}

func (d *DataPoint) SetEmbedding(embedding []float32) {
	d.Embedding = embedding
}

func (d *DataPoint) SetSummary(summary string) {
	d.Summary = summary
}

func (d *DataPoint) ToJSON() ([]byte, error) {
	return json.Marshal(d)
}

func (d *DataPoint) FromJSON(data []byte) error {
	return json.Unmarshal(data, d)
}

func (d *DataPoint) Validate() error {
	if d.ID == "" {
		return fmt.Errorf("DataPoint ID is required")
	}
	if d.Type == "" {
		return fmt.Errorf("DataPoint type is required")
	}
	if d.Content == "" {
		return fmt.Errorf("DataPoint content is required")
	}
	return nil
}

func (d *DataPoint) Clone() *DataPoint {
	clone := &DataPoint{
		ID:         d.ID,
		Type:       d.Type,
		Content:    d.Content,
		Summary:    d.Summary,
		Metadata:   make(map[string]interface{}),
		Source:     d.Source,
		CreatedAt:  d.CreatedAt,
		UpdatedAt:  d.UpdatedAt,
		IsPartOf:   d.IsPartOf,
		Properties: make(map[string]string),
	}

	for k, v := range d.Metadata {
		clone.Metadata[k] = v
	}
	for k, v := range d.Properties {
		clone.Properties[k] = v
	}

	if d.Embedding != nil {
		clone.Embedding = make([]float32, len(d.Embedding))
		copy(clone.Embedding, d.Embedding)
	}

	return clone
}

type DataPointCollection struct {
	dataPoints []*DataPoint
	index      map[string]int
}

func NewCollection() *DataPointCollection {
	return &DataPointCollection{
		dataPoints: make([]*DataPoint, 0),
		index:      make(map[string]int),
	}
}

func (c *DataPointCollection) Add(dp *DataPoint) {
	c.index[dp.ID] = len(c.dataPoints)
	c.dataPoints = append(c.dataPoints, dp)
}

func (c *DataPointCollection) Get(id string) (*DataPoint, bool) {
	if idx, ok := c.index[id]; ok {
		return c.dataPoints[idx], true
	}
	return nil, false
}

func (c *DataPointCollection) List() []*DataPoint {
	return c.dataPoints
}

func (c *DataPointCollection) Len() int {
	return len(c.dataPoints)
}

func (c *DataPointCollection) Filter(predicate func(*DataPoint) bool) []*DataPoint {
	result := make([]*DataPoint, 0)
	for _, dp := range c.dataPoints {
		if predicate(dp) {
			result = append(result, dp)
		}
	}
	return result
}

func (c *DataPointCollection) FindByType(dpType DataPointType) []*DataPoint {
	return c.Filter(func(dp *DataPoint) bool {
		return dp.Type == dpType
	})
}

func (c *DataPointCollection) FindBySource(sourceName string) []*DataPoint {
	return c.Filter(func(dp *DataPoint) bool {
		return dp.Source.Name == sourceName
	})
}

type ExtractedData struct {
	Entities  []*DataPoint `json:"entities"`
	Relations []*DataPoint `json:"relations"`
	Chunks    []*DataPoint `json:"chunks"`
	Summaries []*DataPoint `json:"summaries"`
}

func NewExtractedData() *ExtractedData {
	return &ExtractedData{
		Entities:  make([]*DataPoint, 0),
		Relations: make([]*DataPoint, 0),
		Chunks:    make([]*DataPoint, 0),
		Summaries: make([]*DataPoint, 0),
	}
}

func (e *ExtractedData) AddEntity(dp *DataPoint) {
	e.Entities = append(e.Entities, dp)
}

func (e *ExtractedData) AddRelation(dp *DataPoint) {
	e.Relations = append(e.Relations, dp)
}

func (e *ExtractedData) AddChunk(dp *DataPoint) {
	e.Chunks = append(e.Chunks, dp)
}

func (e *ExtractedData) AddSummary(dp *DataPoint) {
	e.Summaries = append(e.Summaries, dp)
}

func (e *ExtractedData) All() []*DataPoint {
	all := make([]*DataPoint, 0)
	all = append(all, e.Entities...)
	all = append(all, e.Relations...)
	all = append(all, e.Chunks...)
	all = append(all, e.Summaries...)
	return all
}
