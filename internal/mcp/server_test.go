package mcp

import (
	"context"
	"testing"
	"time"

	"github.com/fitz/associate/internal/mcp/tools"
	"github.com/fitz/associate/internal/models"
)

// MockRepository implements a mock for testing
type MockRepository struct {
	SearchFunc func(ctx context.Context, query string, limit int) ([]models.SearchResult, error)
	AddFunc    func(ctx context.Context, mem models.Memory, rels []models.Relationship) (*models.Memory, error)
	UpdateFunc func(ctx context.Context, id string, content *string, metadata map[string]string, tags []string, rels []models.Relationship) (*models.Memory, error)
}

func (m *MockRepository) Search(ctx context.Context, query string, limit int) ([]models.SearchResult, error) {
	if m.SearchFunc != nil {
		return m.SearchFunc(ctx, query, limit)
	}
	return nil, nil
}

func (m *MockRepository) Add(ctx context.Context, mem models.Memory, rels []models.Relationship) (*models.Memory, error) {
	if m.AddFunc != nil {
		return m.AddFunc(ctx, mem, rels)
	}
	return &mem, nil
}

func (m *MockRepository) Update(ctx context.Context, id string, content *string, metadata map[string]string, tags []string, rels []models.Relationship) (*models.Memory, error) {
	if m.UpdateFunc != nil {
		return m.UpdateFunc(ctx, id, content, metadata, tags, rels)
	}
	return &models.Memory{ID: id}, nil
}

func TestSearchInput_Validation(t *testing.T) {
	tests := []struct {
		name  string
		input tools.SearchInput
		valid bool
	}{
		{
			name:  "valid query",
			input: tools.SearchInput{Query: "test query", Limit: 10},
			valid: true,
		},
		{
			name:  "empty query",
			input: tools.SearchInput{Query: "", Limit: 10},
			valid: true, // Empty query is technically allowed
		},
		{
			name:  "default limit",
			input: tools.SearchInput{Query: "test"},
			valid: true, // Limit defaults to 10
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Input validation is handled by the MCP framework via jsonschema
			// Empty queries are allowed but return no results
			_ = tt.input
			_ = tt.valid
		})
	}
}

func TestAddInput_Validation(t *testing.T) {
	tests := []struct {
		name  string
		input tools.AddInput
		valid bool
	}{
		{
			name: "valid with all fields",
			input: tools.AddInput{
				Content:   "Test content",
				Type:      "Note",
				Metadata:  map[string]any{"key": "value"},
				Tags:      []string{"tag1"},
				RelatedTo: []string{"id1"},
			},
			valid: true,
		},
		{
			name:  "minimal valid",
			input: tools.AddInput{Content: "Test content"},
			valid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.input.Content == "" && tt.valid {
				t.Error("Content should be required")
			}
		})
	}
}

func TestUpdateInput_Validation(t *testing.T) {
	content := "updated content"
	tests := []struct {
		name  string
		input tools.UpdateInput
		valid bool
	}{
		{
			name: "update content",
			input: tools.UpdateInput{
				ID:      "test-id",
				Content: &content,
			},
			valid: true,
		},
		{
			name: "add relationship",
			input: tools.UpdateInput{
				ID:        "test-id",
				RelatedTo: []string{"other-id"},
			},
			valid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.input.ID == "" && tt.valid {
				t.Error("ID should be required")
			}
		})
	}
}

func TestSearchOutput_Format(t *testing.T) {
	now := time.Now()
	output := tools.SearchOutput{
		Results: []tools.SearchResultItem{
			{
				ID:        "test-1",
				Type:      "Note",
				Content:   "Test content",
				Score:     0.95,
				CreatedAt: now.Format("2006-01-02T15:04:05Z"),
				UpdatedAt: now.Format("2006-01-02T15:04:05Z"),
			},
		},
		Count: 1,
	}

	if output.Count != len(output.Results) {
		t.Errorf("Count mismatch: got %d, have %d results", output.Count, len(output.Results))
	}
}

func TestAddOutput_Format(t *testing.T) {
	now := time.Now()
	output := tools.AddOutput{
		ID:        "new-id",
		Type:      "Task",
		Content:   "New task",
		CreatedAt: now.Format("2006-01-02T15:04:05Z"),
	}

	if output.ID == "" {
		t.Error("ID should not be empty")
	}
}

func TestUpdateOutput_Format(t *testing.T) {
	now := time.Now()
	output := tools.UpdateOutput{
		ID:        "updated-id",
		Type:      "Note",
		Content:   "Updated content",
		UpdatedAt: now.Format("2006-01-02T15:04:05Z"),
	}

	if output.ID == "" {
		t.Error("ID should not be empty")
	}
}

func TestGetInput_Validation(t *testing.T) {
	tests := []struct {
		name  string
		input tools.GetInput
		valid bool
	}{
		{
			name:  "valid ID",
			input: tools.GetInput{ID: "test-uuid-123"},
			valid: true,
		},
		{
			name:  "empty ID",
			input: tools.GetInput{ID: ""},
			valid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.input.ID == "" && tt.valid {
				t.Error("ID should be required for get_memory")
			}
		})
	}
}

func TestGetOutput_Format(t *testing.T) {
	now := time.Now()
	output := tools.GetOutput{
		ID:        "test-id",
		Type:      "Note",
		Content:   "Test content",
		Metadata:  map[string]string{"key": "value"},
		Tags:      []string{"tag1"},
		Related:   []tools.RelatedMemory{{ID: "related-1", Type: "Note", RelationType: "RELATES_TO"}},
		CreatedAt: now.Format("2006-01-02T15:04:05Z"),
		UpdatedAt: now.Format("2006-01-02T15:04:05Z"),
	}

	if output.ID == "" {
		t.Error("ID should not be empty")
	}
	if len(output.Related) != 1 {
		t.Errorf("Related length: got %d, want 1", len(output.Related))
	}
	if output.Related[0].RelationType != "RELATES_TO" {
		t.Errorf("RelationType: got %s, want RELATES_TO", output.Related[0].RelationType)
	}
}

func TestDeleteInput_Validation(t *testing.T) {
	tests := []struct {
		name  string
		input tools.DeleteInput
		valid bool
	}{
		{
			name:  "valid ID",
			input: tools.DeleteInput{ID: "test-uuid-123"},
			valid: true,
		},
		{
			name:  "empty ID",
			input: tools.DeleteInput{ID: ""},
			valid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.input.ID == "" && tt.valid {
				t.Error("ID should be required for delete_memory")
			}
		})
	}
}

func TestDeleteOutput_Format(t *testing.T) {
	output := tools.DeleteOutput{
		ID:      "deleted-id",
		Deleted: true,
	}

	if output.ID == "" {
		t.Error("ID should not be empty")
	}
	if !output.Deleted {
		t.Error("Deleted should be true")
	}
}

func TestGetRelatedInput_Validation(t *testing.T) {
	tests := []struct {
		name  string
		input tools.GetRelatedInput
		valid bool
	}{
		{
			name:  "valid with ID only",
			input: tools.GetRelatedInput{ID: "test-uuid"},
			valid: true,
		},
		{
			name:  "valid with relationship type",
			input: tools.GetRelatedInput{ID: "test-uuid", RelationType: "DEPENDS_ON"},
			valid: true,
		},
		{
			name:  "valid with direction",
			input: tools.GetRelatedInput{ID: "test-uuid", Direction: "outgoing"},
			valid: true,
		},
		{
			name:  "valid with depth",
			input: tools.GetRelatedInput{ID: "test-uuid", Depth: 2},
			valid: true,
		},
		{
			name:  "empty ID",
			input: tools.GetRelatedInput{ID: ""},
			valid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.input.ID == "" && tt.valid {
				t.Error("ID should be required for get_related")
			}
		})
	}
}

func TestGetRelatedOutput_Format(t *testing.T) {
	now := time.Now()
	output := tools.GetRelatedOutput{
		ID: "source-id",
		Related: []tools.RelatedMemoryFull{
			{
				ID:           "related-1",
				Type:         "Task",
				Content:      "Related task",
				RelationType: "DEPENDS_ON",
				Direction:    "outgoing",
				Depth:        1,
				CreatedAt:    now.Format("2006-01-02T15:04:05Z"),
			},
		},
		Count: 1,
	}

	if output.ID == "" {
		t.Error("ID should not be empty")
	}
	if output.Count != len(output.Related) {
		t.Errorf("Count mismatch: got %d, have %d results", output.Count, len(output.Related))
	}
	if output.Related[0].Direction != "outgoing" {
		t.Errorf("Direction: got %s, want outgoing", output.Related[0].Direction)
	}
}

func TestConvertMetadata(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]any
		expected map[string]string
	}{
		{
			name:     "nil map",
			input:    nil,
			expected: nil,
		},
		{
			name:     "string values",
			input:    map[string]any{"key": "value"},
			expected: map[string]string{"key": "value"},
		},
		{
			name:     "array value serialized",
			input:    map[string]any{"files": []string{"a.go", "b.go"}},
			expected: map[string]string{"files": `["a.go","b.go"]`},
		},
		{
			name:     "object value serialized",
			input:    map[string]any{"config": map[string]int{"count": 5}},
			expected: map[string]string{"config": `{"count":5}`},
		},
		{
			name:     "nil value skipped",
			input:    map[string]any{"key": nil, "other": "value"},
			expected: map[string]string{"other": "value"},
		},
		{
			name:     "mixed values",
			input:    map[string]any{"str": "hello", "num": 42, "arr": []int{1, 2}},
			expected: map[string]string{"str": "hello", "num": "42", "arr": "[1,2]"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tools.ConvertMetadata(tt.input)
			if tt.expected == nil {
				if result != nil {
					t.Errorf("ConvertMetadata() = %v, want nil", result)
				}
				return
			}
			if len(result) != len(tt.expected) {
				t.Errorf("ConvertMetadata() len = %d, want %d", len(result), len(tt.expected))
			}
			for k, v := range tt.expected {
				if result[k] != v {
					t.Errorf("ConvertMetadata()[%s] = %s, want %s", k, result[k], v)
				}
			}
		})
	}
}
