package tools

import (
	"context"
	"fmt"

	"github.com/fitz/associate/internal/graph"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// ReorderTasksInput defines the input for the reorder_tasks tool.
type ReorderTasksInput struct {
	PlanID       string   `json:"plan_id" jsonschema:"required,The ID of the plan containing the tasks to reorder"`
	TaskIDs      []string `json:"task_ids" jsonschema:"required,IDs of tasks to reorder (in the desired new order)"`
	AfterTaskID  *string  `json:"after_task_id,omitempty" jsonschema:"ID of task to position the reordered tasks after. If not specified, tasks are positioned at the start of the plan."`
	BeforeTaskID *string  `json:"before_task_id,omitempty" jsonschema:"ID of task to position the reordered tasks before. Takes precedence for upper bound if both after and before are specified."`
}

// TaskWithPosition contains task info with its position.
type TaskWithPosition struct {
	ID       string  `json:"id"`
	Content  string  `json:"content"`
	Position float64 `json:"position"`
}

// ReorderTasksOutput defines the output for the reorder_tasks tool.
type ReorderTasksOutput struct {
	Tasks []TaskWithPosition `json:"tasks"`
}

// ReorderTasksTool returns the tool definition for reorder_tasks.
func ReorderTasksTool() *mcp.Tool {
	return &mcp.Tool{
		Name:        "reorder_tasks",
		Description: "Reorder tasks within a plan. Moves the specified tasks to new positions relative to other tasks. Tasks are placed in the order provided in task_ids. Use after_task_id and/or before_task_id to specify where to place the group.",
	}
}

// HandleReorderTasks handles the reorder_tasks tool call.
func (h *Handler) HandleReorderTasks(ctx context.Context, req *mcp.CallToolRequest, input ReorderTasksInput) (*mcp.CallToolResult, ReorderTasksOutput, error) {
	h.Logger.Info("reorder_tasks", "plan_id", input.PlanID, "task_count", len(input.TaskIDs))

	if input.PlanID == "" {
		return nil, ReorderTasksOutput{}, fmt.Errorf("plan_id is required")
	}
	if len(input.TaskIDs) == 0 {
		return nil, ReorderTasksOutput{}, fmt.Errorf("task_ids is required and must not be empty")
	}

	// Get all tasks in the plan to validate and get current positions
	_, planTasks, err := h.PlanRepo.GetWithTasks(ctx, input.PlanID)
	if err != nil {
		h.Logger.Error("reorder_tasks failed to get plan", "error", err)
		return nil, ReorderTasksOutput{}, fmt.Errorf("failed to get plan: %w", err)
	}

	// Build a map of task IDs to their info
	taskMap := make(map[string]TaskWithPosition)
	taskPositions := make(map[string]float64)
	for _, t := range planTasks {
		taskMap[t.Task.ID] = TaskWithPosition{
			ID:       t.Task.ID,
			Content:  t.Task.Content,
			Position: t.Position,
		}
		taskPositions[t.Task.ID] = t.Position
	}

	// Validate all task IDs exist in the plan
	for _, taskID := range input.TaskIDs {
		if _, exists := taskMap[taskID]; !exists {
			return nil, ReorderTasksOutput{}, fmt.Errorf("task %s not found in plan %s", taskID, input.PlanID)
		}
	}

	// Calculate anchor positions for the reorder
	var afterPos, beforePos float64

	if input.AfterTaskID != nil && *input.AfterTaskID != "" {
		if pos, exists := taskPositions[*input.AfterTaskID]; exists {
			afterPos = pos
		} else {
			return nil, ReorderTasksOutput{}, fmt.Errorf("after_task_id %s not found in plan", *input.AfterTaskID)
		}
	}

	if input.BeforeTaskID != nil && *input.BeforeTaskID != "" {
		if pos, exists := taskPositions[*input.BeforeTaskID]; exists {
			beforePos = pos
		} else {
			return nil, ReorderTasksOutput{}, fmt.Errorf("before_task_id %s not found in plan", *input.BeforeTaskID)
		}
	}

	// If neither anchor is specified, find the bounds:
	// - If inserting at start (no afterTaskID), find the minimum position
	// - If inserting at end (no beforeTaskID), find the maximum position
	if input.AfterTaskID == nil && input.BeforeTaskID == nil {
		// Find minimum position for tasks not being moved
		minPos := float64(0)
		for id, pos := range taskPositions {
			if !contains(input.TaskIDs, id) && pos < minPos {
				minPos = pos
			}
		}
		beforePos = minPos
	} else if input.AfterTaskID != nil && input.BeforeTaskID == nil {
		// Find the task immediately after the afterTaskID
		afterPosition := afterPos
		nextPos := float64(0)
		for id, pos := range taskPositions {
			if !contains(input.TaskIDs, id) && pos > afterPosition {
				if nextPos == 0 || pos < nextPos {
					nextPos = pos
				}
			}
		}
		beforePos = nextPos // Could be 0 if inserting at end
	} else if input.AfterTaskID == nil && input.BeforeTaskID != nil {
		// Find the task immediately before the beforeTaskID
		beforePosition := beforePos
		prevPos := float64(0)
		for id, pos := range taskPositions {
			if !contains(input.TaskIDs, id) && pos < beforePosition {
				if pos > prevPos {
					prevPos = pos
				}
			}
		}
		afterPos = prevPos // Could be 0 if inserting at start
	}

	// Calculate new positions for the tasks
	positions := graph.CalculateInsertPositions(afterPos, beforePos, len(input.TaskIDs))

	// Build the update map
	newPositions := make(map[string]float64)
	for i, taskID := range input.TaskIDs {
		newPositions[taskID] = positions[i]
	}

	// Update positions in the database
	if err := h.TaskRepo.UpdatePositions(ctx, input.PlanID, newPositions); err != nil {
		h.Logger.Error("reorder_tasks failed to update positions", "error", err)
		return nil, ReorderTasksOutput{}, fmt.Errorf("failed to update positions: %w", err)
	}

	// Build output with new positions
	var outputTasks []TaskWithPosition
	for i, taskID := range input.TaskIDs {
		original := taskMap[taskID]
		outputTasks = append(outputTasks, TaskWithPosition{
			ID:       original.ID,
			Content:  original.Content,
			Position: positions[i],
		})
	}

	h.Logger.Info("reorder_tasks complete", "plan_id", input.PlanID, "tasks_reordered", len(outputTasks))
	return nil, ReorderTasksOutput{Tasks: outputTasks}, nil
}

// contains checks if a string is in a slice
func contains(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}
