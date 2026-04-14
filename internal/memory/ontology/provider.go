package ontology

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"agent-memory/internal/memory/types"
)

type Provider interface {
	Load(ctx context.Context, source string) (*Ontology, error)
	Query(ctx context.Context, ontology *Ontology, query string) ([]Concept, error)
	Link(ctx context.Context, entity *types.Entity, ontology *Ontology) ([]OntologyLink, error)
}

type RDFProvider struct {
	client *http.Client
}

func NewRDFProvider() *RDFProvider {
	return &RDFProvider{
		client: &http.Client{},
	}
}

type Ontology struct {
	ID          string     `json:"id"`
	Name        string     `json:"name"`
	Description string     `json:"description"`
	Source      string     `json:"source"`
	Concepts    []Concept  `json:"concepts"`
	Relations   []Relation `json:"relations"`
	Namespaces  []string   `json:"namespaces"`
	LoadedAt    string     `json:"loaded_at"`
}

type Concept struct {
	ID         string                 `json:"id"`
	Name       string                 `json:"name"`
	Label      string                 `json:"label"`
	Comment    string                 `json:"comment,omitempty"`
	ClassType  string                 `json:"class_type"`
	Properties map[string]interface{} `json:"properties,omitempty"`
	Synonyms   []string               `json:"synonyms,omitempty"`
	Parents    []string               `json:"parents,omitempty"`
	Children   []string               `json:"children,omitempty"`
}

type Relation struct {
	ID        string `json:"id"`
	From      string `json:"from"`
	To        string `json:"to"`
	Relation  string `json:"relation"`
	Label     string `json:"label"`
	Inverse   string `json:"inverse,omitempty"`
	Symmetric bool   `json:"symmetric,omitempty"`
}

type OntologyLink struct {
	ConceptID   string                 `json:"concept_id"`
	ConceptName string                 `json:"concept_name"`
	MatchScore  float64                `json:"match_score"`
	LinkType    string                 `json:"link_type"`
	Properties  map[string]interface{} `json:"properties,omitempty"`
}

func (p *RDFProvider) Load(ctx context.Context, source string) (*Ontology, error) {
	if strings.HasPrefix(source, "http://") || strings.HasPrefix(source, "https://") {
		return p.loadFromURL(ctx, source)
	}
	if strings.HasSuffix(source, ".rdf") || strings.HasSuffix(source, ".xml") {
		return p.loadFromFile(source)
	}
	return nil, fmt.Errorf("unsupported ontology source: %s", source)
}

func (p *RDFProvider) loadFromURL(ctx context.Context, url string) (*Ontology, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch ontology: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, url)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	return p.parseRDF(body)
}

func (p *RDFProvider) loadFromFile(path string) (*Ontology, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}
	return p.parseRDF(content)
}

func (p *RDFProvider) parseRDF(data []byte) (*Ontology, error) {
	var rdf RDFDocument
	if err := xml.Unmarshal(data, &rdf); err != nil {
		return nil, fmt.Errorf("parse RDF: %w", err)
	}

	ontology := &Ontology{
		ID:         rdf.RDF.About,
		Concepts:   []Concept{},
		Relations:  []Relation{},
		Namespaces: []string{},
	}

	for prefix, ns := range rdf.RDF.Namespaces {
		ontology.Namespaces = append(ontology.Namespaces, prefix+":"+ns)
	}

	for _, desc := range rdf.RDF.Descriptions {
		concept := Concept{
			ID:         desc.About,
			Name:       desc.GetLabel(),
			Label:      desc.GetLabel(),
			Comment:    desc.Comment,
			Properties: make(map[string]interface{}),
		}

		for _, prop := range desc.Properties {
			if prop.Resource != "" {
				concept.Properties[prop.Name] = prop.Resource
			} else if len(prop.Literals) > 0 {
				concept.Properties[prop.Name] = prop.Literals[0].Data
			}
		}

		if desc.Type != "" {
			concept.ClassType = desc.Type
		}

		ontology.Concepts = append(ontology.Concepts, concept)
	}

	for _, seq := range rdf.RDF.Seqs {
		for _, item := range seq.Items {
			if item.Resource != "" {
				ontology.Relations = append(ontology.Relations, Relation{
					From:     seq.About,
					To:       item.Resource,
					Relation: "item",
					Label:    "has item",
				})
			}
		}
	}

	return ontology, nil
}

func (p *RDFProvider) Query(ctx context.Context, ontology *Ontology, query string) ([]Concept, error) {
	lowerQuery := strings.ToLower(query)
	var results []Concept

	for _, concept := range ontology.Concepts {
		if strings.Contains(strings.ToLower(concept.Name), lowerQuery) {
			results = append(results, concept)
			continue
		}
		if strings.Contains(strings.ToLower(concept.Label), lowerQuery) {
			results = append(results, concept)
			continue
		}
		if strings.Contains(strings.ToLower(concept.Comment), lowerQuery) {
			results = append(results, concept)
			continue
		}
		for _, syn := range concept.Synonyms {
			if strings.Contains(strings.ToLower(syn), lowerQuery) {
				results = append(results, concept)
				break
			}
		}
	}

	return results, nil
}

func (p *RDFProvider) Link(ctx context.Context, entity *types.Entity, ontology *Ontology) ([]OntologyLink, error) {
	var links []OntologyLink

	entityLower := strings.ToLower(entity.Name)

	for _, concept := range ontology.Concepts {
		var score float64
		var linkType string

		if strings.ToLower(concept.Name) == entityLower {
			score = 1.0
			linkType = "exact_match"
		} else if strings.Contains(strings.ToLower(concept.Name), entityLower) ||
			strings.Contains(entityLower, strings.ToLower(concept.Name)) {
			score = 0.8
			linkType = "partial_match"
		} else {
			for _, syn := range concept.Synonyms {
				if strings.ToLower(syn) == entityLower {
					score = 0.9
					linkType = "synonym_match"
					break
				}
			}
		}

		if score > 0 {
			links = append(links, OntologyLink{
				ConceptID:   concept.ID,
				ConceptName: concept.Name,
				MatchScore:  score,
				LinkType:    linkType,
				Properties:  concept.Properties,
			})
		}
	}

	return links, nil
}

type RDFDocument struct {
	RDF RDFRoot `xml:"RDF"`
}

type RDFRoot struct {
	XMLName      xml.Name          `xml:"RDF"`
	About        string            `xml:"about,attr"`
	Namespaces   map[string]string `xml:"-"`
	Descriptions []RDFDescription  `xml:"Description"`
	Seqs         []RDFSeq          `xml:"Seq"`
}

type RDFDescription struct {
	XMLName    xml.Name      `xml:"Description"`
	About      string        `xml:"about,attr"`
	Type       string        `xml:"type,attr"`
	Label      string        `xml:"label,attr"`
	Comment    string        `xml:"comment"`
	Properties []RDFProperty `xml:",any"`
}

func (d *RDFDescription) GetLabel() string {
	if d.Label != "" {
		return d.Label
	}
	if d.Properties != nil {
		for _, p := range d.Properties {
			if p.Name == "label" || p.Name == "rdfs:label" || p.Name == "skos:prefLabel" {
				return p.Literals[0].Data
			}
		}
	}
	return d.About
}

type RDFProperty struct {
	Name     string       `xml:"name,attr"`
	Resource string       `xml:"resource,attr"`
	Literals []RDFLiteral `xml:",chardata"`
}

type RDFLiteral struct {
	Data string `xml:",chardata"`
}

type RDFSeq struct {
	XMLName xml.Name  `xml:"Seq"`
	About   string    `xml:"about,attr"`
	Items   []RDFItem `xml:"li"`
}

type RDFItem struct {
	Resource string `xml:"resource,attr"`
}

type OWLProvider struct {
	rdfProvider *RDFProvider
}

func NewOWLProvider() *OWLProvider {
	return &OWLProvider{
		rdfProvider: NewRDFProvider(),
	}
}

func (p *OWLProvider) Load(ctx context.Context, source string) (*Ontology, error) {
	onto, err := p.rdfProvider.loadFromURL(ctx, source)
	if err != nil {
		return nil, err
	}

	for i := range onto.Concepts {
		if onto.Concepts[i].ClassType == "" {
			onto.Concepts[i].ClassType = "owl:Class"
		}
	}

	return onto, nil
}

func (p *OWLProvider) Query(ctx context.Context, ontology *Ontology, query string) ([]Concept, error) {
	return p.rdfProvider.Query(ctx, ontology, query)
}

func (p *OWLProvider) Link(ctx context.Context, entity *types.Entity, ontology *Ontology) ([]OntologyLink, error) {
	return p.rdfProvider.Link(ctx, entity, ontology)
}

type SKOSProvider struct {
	rdfProvider *RDFProvider
}

func NewSKOSProvider() *SKOSProvider {
	return &SKOSProvider{
		rdfProvider: NewRDFProvider(),
	}
}

func (p *SKOSProvider) Load(ctx context.Context, source string) (*Ontology, error) {
	onto, err := p.rdfProvider.loadFromURL(ctx, source)
	if err != nil {
		return nil, err
	}

	for i := range onto.Concepts {
		onto.Concepts[i].ClassType = "skos:Concept"
	}

	return onto, nil
}

func (p *SKOSProvider) Query(ctx context.Context, ontology *Ontology, query string) ([]Concept, error) {
	return p.rdfProvider.Query(ctx, ontology, query)
}

func (p *SKOSProvider) Link(ctx context.Context, entity *types.Entity, ontology *Ontology) ([]OntologyLink, error) {
	return p.rdfProvider.Link(ctx, entity, ontology)
}
