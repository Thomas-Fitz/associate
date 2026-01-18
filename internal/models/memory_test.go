package models

import (
	"testing"
	"time"
)

func TestMemoryType_Constants(t *testing.T) {
	// Verify all expected types are defined
	// Note: TypeTask and TypeProject have been moved to separate Task and Plan node types
	types := []MemoryType{TypeNote, TypeRepository, TypeGeneral}
	expected := []string{"Note", "Repository", "Memory"}

	for i, mt := range types {
		if string(mt) != expected[i] {
			t.Errorf("MemoryType %d: got %s, want %s", i, mt, expected[i])
		}
	}
}

func TestRelationType_Constants(t *testing.T) {
	// Verify all expected relationship types
	types := []RelationType{RelRelatesTo, RelPartOf, RelReferences, RelDependsOn, RelBlocks, RelFollows, RelImplements}
	expected := []string{"RELATES_TO", "PART_OF", "REFERENCES", "DEPENDS_ON", "BLOCKS", "FOLLOWS", "IMPLEMENTS"}

	for i, rt := range types {
		if string(rt) != expected[i] {
			t.Errorf("RelationType %d: got %s, want %s", i, rt, expected[i])
		}
	}
}

func TestMemory_Struct(t *testing.T) {
	now := time.Now()
	mem := Memory{
		ID:        "test-id",
		Type:      TypeNote,
		Content:   "Test content",
		Metadata:  map[string]string{"key": "value"},
		Tags:      []string{"tag1", "tag2"},
		CreatedAt: now,
		UpdatedAt: now,
	}

	if mem.ID != "test-id" {
		t.Errorf("ID: got %s, want test-id", mem.ID)
	}
	if mem.Type != TypeNote {
		t.Errorf("Type: got %s, want %s", mem.Type, TypeNote)
	}
	if mem.Content != "Test content" {
		t.Errorf("Content: got %s, want 'Test content'", mem.Content)
	}
	if len(mem.Tags) != 2 {
		t.Errorf("Tags length: got %d, want 2", len(mem.Tags))
	}
}

func TestSearchResult_Struct(t *testing.T) {
	sr := SearchResult{
		Memory:  Memory{ID: "test"},
		Score:   0.95,
		Related: []string{"related-1", "related-2"},
	}

	if sr.Score != 0.95 {
		t.Errorf("Score: got %f, want 0.95", sr.Score)
	}
	if len(sr.Related) != 2 {
		t.Errorf("Related length: got %d, want 2", len(sr.Related))
	}
}

func TestRelatedInfo_Struct(t *testing.T) {
	ri := RelatedInfo{
		ID:           "related-id",
		Type:         TypeNote,
		RelationType: "DEPENDS_ON",
		Direction:    "outgoing",
	}

	if ri.ID != "related-id" {
		t.Errorf("ID: got %s, want related-id", ri.ID)
	}
	if ri.Type != TypeNote {
		t.Errorf("Type: got %s, want %s", ri.Type, TypeNote)
	}
	if ri.RelationType != "DEPENDS_ON" {
		t.Errorf("RelationType: got %s, want DEPENDS_ON", ri.RelationType)
	}
	if ri.Direction != "outgoing" {
		t.Errorf("Direction: got %s, want outgoing", ri.Direction)
	}
}

func TestRelatedMemoryResult_Struct(t *testing.T) {
	now := time.Now()
	rmr := RelatedMemoryResult{
		Memory: Memory{
			ID:        "mem-id",
			Type:      TypeNote,
			Content:   "Test content",
			CreatedAt: now,
			UpdatedAt: now,
		},
		RelationType: "BLOCKS",
		Direction:    "incoming",
		Depth:        2,
	}

	if rmr.Memory.ID != "mem-id" {
		t.Errorf("Memory.ID: got %s, want mem-id", rmr.Memory.ID)
	}
	if rmr.RelationType != "BLOCKS" {
		t.Errorf("RelationType: got %s, want BLOCKS", rmr.RelationType)
	}
	if rmr.Direction != "incoming" {
		t.Errorf("Direction: got %s, want incoming", rmr.Direction)
	}
	if rmr.Depth != 2 {
		t.Errorf("Depth: got %d, want 2", rmr.Depth)
	}
}
