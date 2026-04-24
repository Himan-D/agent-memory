package ontology

import (
	"context"
	"log"
	"strings"
	"time"

	"agent-memory/internal/memory/types"
)

var DefaultOntologySources = []string{
	"https://dbpedia.org/data/dbpedia_onto.rdf",
	"https://schema.org/version/latest/schemaorg.rdf",
}

type Loader struct {
	provider Provider
	cache    map[string]*Ontology
}

func NewLoader(provider Provider) *Loader {
	return &Loader{
		provider: provider,
		cache:    make(map[string]*Ontology),
	}
}

func (l *Loader) LoadDefault(ctx context.Context) ([]*Ontology, error) {
	sources := DefaultOntologySources
	var ontologies []*Ontology

	for _, source := range sources {
		onto, err := l.Load(ctx, source)
		if err != nil {
			log.Printf("Warning: failed to load ontology %s: %v", source, err)
			continue
		}
		ontologies = append(ontologies, onto)
		log.Printf("Loaded ontology: %s with %d concepts", onto.Name, len(onto.Concepts))
	}

	return ontologies, nil
}

func (l *Loader) Load(ctx context.Context, source string) (*Ontology, error) {
	if onto, ok := l.cache[source]; ok {
		return onto, nil
	}

	onto, err := l.provider.Load(ctx, source)
	if err != nil {
		return nil, err
	}

	onto.LoadedAt = time.Now().Format(time.RFC3339)
	l.cache[source] = onto

	return onto, nil
}

func (l *Loader) LinkEntity(ctx context.Context, entity *types.Entity, ontologies []*Ontology) []OntologyLink {
	var allLinks []OntologyLink

	for _, onto := range ontologies {
		links, err := l.provider.Link(ctx, entity, onto)
		if err != nil {
			continue
		}
		allLinks = append(allLinks, links...)
	}

	return allLinks
}

func (l *Loader) DetectConflicts(entityName string, ontologies []*Ontology, existingMemories []*types.Memory) []Conflict {
	var conflicts []Conflict

	for _, memory := range existingMemories {
		memoryLower := strings.ToLower(memory.Content)
		entityLower := strings.ToLower(entityName)

		for _, onto := range ontologies {
			for _, relation := range onto.Relations {
				if !relation.Symmetric {
					continue
				}

				relLower := strings.ToLower(relation.Relation)
				if relLower != "contradicts" && relLower != "opposite" && relLower != "inverse" {
					continue
				}

				fromLower := strings.ToLower(relation.From)
				toLower := strings.ToLower(relation.To)

				if (strings.Contains(memoryLower, fromLower) && strings.Contains(entityLower, toLower)) ||
					(strings.Contains(memoryLower, toLower) && strings.Contains(entityLower, fromLower)) {
					conflicts = append(conflicts, Conflict{
						MemoryA:    memory.ID,
						MemoryB:    entityName,
						Type:       relation.Relation,
						Confidence: 0.8,
					})
				}
			}
		}
	}

	return conflicts
}

type Conflict struct {
	MemoryA    string  `json:"memory_a"`
	MemoryB    string  `json:"memory_b"`
	Type       string  `json:"type"`
	Confidence float64 `json:"confidence"`
}