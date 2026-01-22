---
name: mcp-manual-tests
description: Trigger Associate MCP tools and check results.
---

# Associate MCP Server Manual Testing

**IMPORTANT: You have direct access to MCP tools as functions. Call them directly - do NOT use HTTP, curl, scripts, or any other method.**

## How to Run Tests

The MCP tools are available to you as callable functions with the `mcp_associate_` prefix:

- `mcp_associate_add_memory` - Add a memory
- `mcp_associate_get_memory` - Get a memory by ID  
- `mcp_associate_search_memories` - Search memories
- `mcp_associate_update_memory` - Update a memory
- `mcp_associate_delete_memory` - Delete a memory
- `mcp_associate_create_plan` - Create a plan
- `mcp_associate_get_plan` - Get a plan by ID
- `mcp_associate_list_plans` - List plans
- `mcp_associate_update_plan` - Update a plan
- `mcp_associate_delete_plan` - Delete a plan
- `mcp_associate_create_task` - Create a task
- `mcp_associate_get_task` - Get a task by ID
- `mcp_associate_list_tasks` - List tasks
- `mcp_associate_update_task` - Update a task
- `mcp_associate_delete_task` - Delete a task
- `mcp_associate_get_related` - Get related nodes
- `mcp_associate_reorder_tasks` - Reorder tasks within a plan

**Call these tools directly like any other tool. Do NOT search for scripts, curl endpoints, or inspect code.**

## Test Sequence

Generate a unique test token first: `mcp-test-<current_timestamp>`

### Phase 1: Memory Lifecycle

1. Call `mcp_associate_add_memory` with content containing token, tags=["mcp-test"], type="Note"
   - Extract the `id` from response
2. Call `mcp_associate_get_memory` with extracted id - verify content matches
3. Call `mcp_associate_search_memories` with the token as query - verify memory found
4. Call `mcp_associate_update_memory` with extracted id, new content with " - updated" appended
5. Call `mcp_associate_get_memory` with extracted id - verify update applied

### Phase 2: Plan Lifecycle

1. Call `mcp_associate_create_plan` with name containing token, description="Test plan"
   - Extract the `id` from response (this is the plan_id)
2. Call `mcp_associate_get_plan` with extracted plan_id - verify name/description
3. Call `mcp_associate_list_plans` with status="active" - verify plan appears
4. Call `mcp_associate_update_plan` with extracted plan_id, new description="Updated description"
5. Call `mcp_associate_get_plan` with extracted plan_id - verify update applied

### Phase 3: Task Lifecycle and Ordering

1. Call `mcp_associate_create_task` with content containing token, plan_ids=[extracted plan_id]
   - Extract the `id` from response (this is the task_id_1)
2. Call `mcp_associate_get_task` with extracted task_id_1 - verify content and plan linkage
3. Call `mcp_associate_list_tasks` with plan_id filter - verify task appears and **has position property**
4. Call `mcp_associate_create_task` second task with same plan_id - extract task_id_2
5. Call `mcp_associate_list_tasks` again - verify both tasks appear ordered by position
6. Call `mcp_associate_get_plan` with plan_id - verify tasks in response have position, depends_on, blocks fields
7. Call `mcp_associate_create_task` with `after_task_id=task_id_1` - insert between tasks, extract task_id_3
8. Call `mcp_associate_list_tasks` with plan_id - verify ordering: task_id_1, task_id_3, task_id_2
9. Call `mcp_associate_update_task` with task_id_1, status="completed"
10. Call `mcp_associate_get_task` with task_id_1 - verify status updated

### Phase 4: Reorder Tasks

1. Call `mcp_associate_reorder_tasks` with plan_id, task_ids=[task_id_2, task_id_1, task_id_3]
   - Verify response contains tasks with new positions
2. Call `mcp_associate_list_tasks` with plan_id - verify ordering changed to: task_id_2, task_id_1, task_id_3
3. Call `mcp_associate_reorder_tasks` with task_ids=[task_id_3, task_id_1], before_task_id=task_id_2
   - Verify positioning works with before_task_id parameter

### Phase 5: Relationship Tests

1. Call `mcp_associate_get_related` with plan_id - should show all tasks with PART_OF relationship
2. Call `mcp_associate_get_related` with task_id_1 - should show the plan with PART_OF relationship

### Phase 6: Cleanup

1. Call `mcp_associate_delete_task` with task_id_1 - verify deleted=true
2. Call `mcp_associate_delete_plan` with plan_id - verify deleted=true, tasks_deleted=2 (remaining tasks)
3. Call `mcp_associate_delete_memory` with memory_id - verify deleted=true

### Phase 7: Edge Cases for Task Ordering

1. **Task position in list_tasks without plan filter**:
   - Call `mcp_associate_list_tasks` without plan_id - verify position is NOT included (null/omitted)

2. **Position increments correctly**:
   - Create new plan (plan_id_2)
   - Create task_a (should have position=1000)
   - Create task_b (should have position=2000)
   - Create task_c (should have position=3000)
   - Verify positions are DefaultPositionIncrement (1000) apart

3. **Insert before first task**:
   - Create new plan (plan_id_3)
   - Create task_1 (position=1000)
   - Create task_2 (position=2000)
   - Create task_0 with before_task_id=task_1 - should have position < 1000
   - List tasks - verify order: task_0, task_1, task_2

4. **Insert after last task**:
   - Create task_3 with after_task_id=task_2 - should have position > 2000
   - List tasks - verify order: task_1, task_2, task_3

5. **Insert between two tasks**:
   - Create task_1_5 with after_task_id=task_1 and before_task_id=task_2 - should be between them
   - List tasks - verify order: task_1, task_1_5, task_2

6. **Reorder single task**:
   - Call `mcp_associate_reorder_tasks` with single task_id - should succeed

7. **Delete plan with tasks**:
   - Call `mcp_associate_delete_plan` with plan_id_3
   - Verify response has tasks_deleted > 0

## Report Format

For each tool call, report:
- **Tool**: The tool name called
- **Input**: Parameters passed
- **Response**: Key fields from response
- **Result**: PASS or FAIL
- **Notes**: Any specific assertions checked (e.g., "position=1000", "position not included")

## Task Ordering Specific Checks

For `list_tasks`:
- ✅ Must return `position` field when `plan_id` is provided
- ✅ Must NOT return `position` field when no `plan_id` provided
- ✅ Tasks must be ordered by position ASC

For `get_plan`:
- ✅ Tasks must include `position` field (float64)
- ✅ Tasks must include `depends_on` field (array or null)
- ✅ Tasks must include `blocks` field (array or null)
- ✅ Tasks must be ordered by position ASC

For `create_task`:
- ✅ Without positioning params - position = max_position + 1000
- ✅ With `after_task_id` - position between after task and next task
- ✅ With `before_task_id` - position between before task and previous task
- ✅ With both params - position between specified tasks

For `reorder_tasks`:
- ✅ Without `before_task_id` or `after_task_id` - appends to end
- ✅ With `after_task_id` - positions after specified task
- ✅ With `before_task_id` - positions before specified task
- ✅ Multiple tasks maintain relative order

## Important Notes

- Run tests **sequentially** - each step depends on IDs from previous steps
- Extract the `id` field from each create response before proceeding
- The `plan_ids` parameter for create_task must be an array: `["<plan_id>"]`
- Position values are stored as float64 to allow fine-grained ordering
- DefaultPositionIncrement is 1000.0 for spacing new tasks
