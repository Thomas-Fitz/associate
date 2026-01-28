package graph

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/Thomas-Fitz/associate/internal/models"
)

// ValidRelationTypes is the complete set of allowed relationship type names.
var ValidRelationTypes = map[models.RelationType]bool{
	models.RelRelatesTo:  true,
	models.RelPartOf:     true,
	models.RelReferences: true,
	models.RelDependsOn:  true,
	models.RelBlocks:     true,
	models.RelFollows:    true,
	models.RelImplements: true,
	models.RelBelongsTo:  true,
}

// ValidateRelationType checks that a relationship type is one of the known constants.
func ValidateRelationType(relType models.RelationType) error {
	if !ValidRelationTypes[relType] {
		return fmt.Errorf("invalid relationship type: %q", relType)
	}
	return nil
}

// EscapeCypherString escapes a string for safe interpolation into Cypher queries.
func EscapeCypherString(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "'", "\\'")
	s = strings.ReplaceAll(s, "\"", "\\\"")
	s = strings.ReplaceAll(s, "\n", "\\n")
	s = strings.ReplaceAll(s, "\r", "\\r")
	s = strings.ReplaceAll(s, "\t", "\\t")
	s = strings.ReplaceAll(s, "\x00", "") // Strip null bytes
	return s
}

// NodeLabelPredicate returns an AGE-compatible label predicate for use in WHERE clauses.
// Apache AGE doesn't support the standard Cypher "node:Label" syntax in WHERE clauses.
// Instead, we use the label() function: label(node) IN ['Memory', 'Plan', 'Task', 'Zone']
func NodeLabelPredicate(nodeVar string) string {
	return fmt.Sprintf("label(%s) IN ['Memory', 'Plan', 'Task', 'Zone']", nodeVar)
}

// tagsToCypherList converts a Go string slice to a Cypher list literal.
func tagsToCypherList(tags []string) string {
	if len(tags) == 0 {
		return "[]"
	}
	escaped := make([]string, len(tags))
	for i, t := range tags {
		escaped[i] = fmt.Sprintf("'%s'", EscapeCypherString(t))
	}
	return "[" + strings.Join(escaped, ", ") + "]"
}

// metadataToJSON converts a metadata map to a JSON string.
func metadataToJSON(m map[string]string) string {
	if len(m) == 0 {
		return ""
	}
	b, err := json.Marshal(m)
	if err != nil {
		return ""
	}
	return string(b)
}

// jsonToMetadata converts a JSON string to a metadata map.
func jsonToMetadata(s string) map[string]string {
	if s == "" {
		return nil
	}
	var m map[string]string
	if err := json.Unmarshal([]byte(s), &m); err != nil {
		return nil
	}
	return m
}

// getString extracts a string property from a map.
func getString(props map[string]interface{}, key string) string {
	if v, ok := props[key].(string); ok {
		return v
	}
	return ""
}

// joinStrings joins strings with a separator.
func joinStrings(strs []string, sep string) string {
	if len(strs) == 0 {
		return ""
	}
	return strings.Join(strs, sep)
}

// parseAGTypeProperties parses the properties from an agtype vertex/edge string.
// AGE returns vertices like: {"id": 12345, "label": "Memory", "properties": {"id": "abc", ...}}::vertex
func parseAGTypeProperties(agtypeStr string) (map[string]interface{}, error) {
	// Remove the ::vertex or ::edge suffix
	re := regexp.MustCompile(`::(?:vertex|edge)$`)
	jsonStr := re.ReplaceAllString(agtypeStr, "")

	var wrapper struct {
		Properties map[string]interface{} `json:"properties"`
	}
	if err := json.Unmarshal([]byte(jsonStr), &wrapper); err != nil {
		return nil, err
	}
	return wrapper.Properties, nil
}

// propsToMemory converts a properties map to a Memory struct.
func propsToMemory(props map[string]interface{}) models.Memory {
	mem := models.Memory{
		ID:      getString(props, "id"),
		Type:    models.MemoryType(getString(props, "type")),
		Content: getString(props, "content"),
	}

	if metaStr := getString(props, "metadata"); metaStr != "" {
		mem.Metadata = jsonToMetadata(metaStr)
	}

	if tagsRaw, ok := props["tags"]; ok {
		if tagsArr, ok := tagsRaw.([]interface{}); ok {
			for _, t := range tagsArr {
				if s, ok := t.(string); ok {
					mem.Tags = append(mem.Tags, s)
				}
			}
		}
	}

	if createdStr := getString(props, "created_at"); createdStr != "" {
		if t, err := time.Parse(time.RFC3339, createdStr); err == nil {
			mem.CreatedAt = t
		}
	}
	if updatedStr := getString(props, "updated_at"); updatedStr != "" {
		if t, err := time.Parse(time.RFC3339, updatedStr); err == nil {
			mem.UpdatedAt = t
		}
	}

	return mem
}

// propsToPlan converts a properties map to a Plan struct.
func propsToPlan(props map[string]interface{}) models.Plan {
	plan := models.Plan{
		ID:          getString(props, "id"),
		Name:        getString(props, "name"),
		Description: getString(props, "description"),
		Status:      models.PlanStatus(getString(props, "status")),
	}

	if metaStr := getString(props, "metadata"); metaStr != "" {
		plan.Metadata = jsonToMetadata(metaStr)
	}

	if tagsRaw, ok := props["tags"]; ok {
		if tagsArr, ok := tagsRaw.([]interface{}); ok {
			for _, t := range tagsArr {
				if s, ok := t.(string); ok {
					plan.Tags = append(plan.Tags, s)
				}
			}
		}
	}

	if createdStr := getString(props, "created_at"); createdStr != "" {
		if t, err := time.Parse(time.RFC3339, createdStr); err == nil {
			plan.CreatedAt = t
		}
	}
	if updatedStr := getString(props, "updated_at"); updatedStr != "" {
		if t, err := time.Parse(time.RFC3339, updatedStr); err == nil {
			plan.UpdatedAt = t
		}
	}

	return plan
}

// propsToTask converts a properties map to a Task struct.
func propsToTask(props map[string]interface{}) models.Task {
	task := models.Task{
		ID:      getString(props, "id"),
		Content: getString(props, "content"),
		Status:  models.TaskStatus(getString(props, "status")),
	}

	if metaStr := getString(props, "metadata"); metaStr != "" {
		task.Metadata = jsonToMetadata(metaStr)
	}

	if tagsRaw, ok := props["tags"]; ok {
		if tagsArr, ok := tagsRaw.([]interface{}); ok {
			for _, t := range tagsArr {
				if s, ok := t.(string); ok {
					task.Tags = append(task.Tags, s)
				}
			}
		}
	}

	if createdStr := getString(props, "created_at"); createdStr != "" {
		if t, err := time.Parse(time.RFC3339, createdStr); err == nil {
			task.CreatedAt = t
		}
	}
	if updatedStr := getString(props, "updated_at"); updatedStr != "" {
		if t, err := time.Parse(time.RFC3339, updatedStr); err == nil {
			task.UpdatedAt = t
		}
	}

	return task
}

// propsToZone converts a properties map to a Zone struct.
func propsToZone(props map[string]interface{}) models.Zone {
	zone := models.Zone{
		ID:          getString(props, "id"),
		Name:        getString(props, "name"),
		Description: getString(props, "description"),
	}

	if metaStr := getString(props, "metadata"); metaStr != "" {
		zone.Metadata = jsonToMetadata(metaStr)
	}

	if tagsRaw, ok := props["tags"]; ok {
		if tagsArr, ok := tagsRaw.([]interface{}); ok {
			for _, t := range tagsArr {
				if s, ok := t.(string); ok {
					zone.Tags = append(zone.Tags, s)
				}
			}
		}
	}

	if createdStr := getString(props, "created_at"); createdStr != "" {
		if t, err := time.Parse(time.RFC3339, createdStr); err == nil {
			zone.CreatedAt = t
		}
	}
	if updatedStr := getString(props, "updated_at"); updatedStr != "" {
		if t, err := time.Parse(time.RFC3339, updatedStr); err == nil {
			zone.UpdatedAt = t
		}
	}

	return zone
}

// toFloat64 converts various numeric types to float64.
func toFloat64(v interface{}) float64 {
	switch val := v.(type) {
	case float64:
		return val
	case float32:
		return float64(val)
	case int64:
		return float64(val)
	case int:
		return float64(val)
	case json.Number:
		if f, err := val.Float64(); err == nil {
			return f
		}
	default:
		return 0
	}
	return 0
}
