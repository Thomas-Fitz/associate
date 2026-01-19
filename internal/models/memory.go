package models

import "time"

// MemoryType defines the category of a memory
type MemoryType string

const (
	TypeNote       MemoryType = "Note"
	TypeRepository MemoryType = "Repository"
	TypeGeneral    MemoryType = "Memory"
)

// RelationType defines the type of relationship between memories
type RelationType string

const (
	RelRelatesTo  RelationType = "RELATES_TO"
	RelPartOf     RelationType = "PART_OF"
	RelReferences RelationType = "REFERENCES"
	RelDependsOn  RelationType = "DEPENDS_ON"
	RelBlocks     RelationType = "BLOCKS"     // Task A blocks Task B (A must complete first)
	RelFollows    RelationType = "FOLLOWS"    // Sequence ordering (A comes after B)
	RelImplements RelationType = "IMPLEMENTS" // Code implements a decision/task
)

// Memory represents a memory node in the graph database.
type Memory struct {
	ID        string            `json:"id"`
	Type      MemoryType        `json:"type"`
	Content   string            `json:"content"`
	Metadata  map[string]string `json:"metadata,omitempty"`
	Tags      []string          `json:"tags,omitempty"`
	CreatedAt time.Time         `json:"created_at"`
	UpdatedAt time.Time         `json:"updated_at"`
}

// Relationship represents a connection between two memories
type Relationship struct {
	FromID string       `json:"from_id"`
	ToID   string       `json:"to_id"`
	Type   RelationType `json:"type"`
}

// SearchResult contains a memory with its relevance score
type SearchResult struct {
	Memory  Memory   `json:"memory"`
	Score   float64  `json:"score"`
	Related []string `json:"related,omitempty"` // IDs of related memories
}

// RelatedInfo contains info about a related memory and its relationship
type RelatedInfo struct {
	ID           string     `json:"id"`
	Type         MemoryType `json:"type"`
	RelationType string     `json:"relationship_type"`
	Direction    string     `json:"direction"` // "incoming" or "outgoing"
}

// RelatedMemoryResult contains full memory data with relationship metadata
type RelatedMemoryResult struct {
	Memory       Memory `json:"memory"`
	RelationType string `json:"relationship_type"`
	Direction    string `json:"direction"`
	Depth        int    `json:"depth"`
}
