# Associate MCP Server Manual Testing

You are testing the Associate MCP server which provides AI agents with graph database memory through Neo4j. Run a comprehensive test of all MCP tools to verify they work correctly.

## Prerequisites
- Docker container is running (`docker-compose up -d`)
- Neo4j is healthy and accessible

## Test Approach
1. Use a unique timestamp token for all test data: `mcp-test-<ISO_TIMESTAMP>`
2. Run tests sequentially, capturing full responses
3. Create ephemeral objects, verify them, update them, then delete them
4. Report PASS/FAIL for each operation with details

## Tools to Test

### Memory Tools
- `add_memory` - Create memory with content, tags, type
- `get_memory` - Retrieve by ID
- `search_memories` - Full-text search (quote special characters)
- `update_memory` - Modify content/tags/relationships
- `delete_memory` - Remove memory
- `get_related` - Traverse graph relationships

### Plan Tools
- `create_plan` - Create with name, description, status
- `get_plan` - Retrieve by ID (includes tasks)
- `list_plans` - List with optional status/tags filter
- `update_plan` - Modify name/description/status
- `delete_plan` - Remove plan (cascade deletes orphan tasks)

### Task Tools
- `create_task` - Create with content, plan_ids (required), status
- `get_task` - Retrieve by ID (includes linked plans)
- `list_tasks` - List with optional plan_id/status filter
- `update_task` - Modify content/status/plan links
- `delete_task` - Remove task

## Test Sequence

### Phase 1: Generate Test Token
```bash
powershell -NoProfile -Command "[DateTime]::UtcNow.ToString('yyyy-MM-ddTHH:mm:ssZ')"
```
Use this timestamp in all test content for easy identification and cleanup.

### Phase 2: Memory Lifecycle
1. `add_memory` with content containing the token, tags=["mcp-test"], type="Note"
2. `get_memory` with returned ID → verify content matches
3. `search_memories` with quoted token → verify memory found
4. `update_memory` → append " — updated" to content
5. `get_memory` → verify update applied
6. (Keep memory for relationship tests)

### Phase 3: Plan Lifecycle
1. `create_plan` with name containing token
2. `get_plan` → verify name/description
3. `list_plans` with status="active" → verify plan appears
4. `update_plan` → change description
5. `get_plan` → verify update applied

### Phase 4: Task Lifecycle
1. `create_task` with content containing token, plan_ids=[plan_id from Phase 3]
2. `get_task` → verify content and plan linkage
3. `list_tasks` with plan_id filter → verify task appears
4. `update_task` → set status="completed"
5. `get_task` → verify status updated

### Phase 5: Relationship Tests
1. `get_related` on the plan → should show the task
2. `get_related` on the task → should show the plan

### Phase 6: Cleanup (Delete in Order)
1. `delete_task` → verify {"deleted": true}
2. `delete_plan` → verify {"deleted": true, "tasks_deleted": 0}
3. `delete_memory` → verify {"deleted": true}

### Phase 7: Edge Case Tests
1. **Delete plan with no tasks**: Create plan → immediately delete (no tasks created)
   - This tests the fix for the UNWIND empty collection bug
2. **Cascade delete**: Create plan → create task → delete plan
   - Verify tasks_deleted > 0 and task is gone
3. **Search with special characters**: Test quoted searches for timestamps with colons

## Test Method

### MCP tools
```bash
(
echo '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"test","version":"1.0"}}}'
sleep 0.5
echo '{"jsonrpc":"2.0","method":"notifications/initialized"}'
sleep 0.5
echo '{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"TOOL_NAME","arguments":{ARGS}}}'
sleep 1
) | docker run -i --rm --network associate_default -e NEO4J_URI=bolt://neo4j:7687 associate-associate 2>&1
```

## Debugging

### Query Neo4j for debugging purposes only:
```bash
docker exec associate-neo4j cypher-shell -u neo4j -p password "MATCH (n) WHERE n:Memory OR n:Plan OR n:Task RETURN labels(n)[0] as type, n.id, n.name, n.content LIMIT 20"
```

## Expected Results

| Tool | Success Criteria |
|------|------------------|
| add_memory | Returns object with `id`, `content` matches input |
| get_memory | Returns object, `content` contains token |
| search_memories | Returns count > 0, results include the memory |
| update_memory | Returns object with updated fields |
| delete_memory | Returns `{"deleted": true, "id": "..."}` |
| create_plan | Returns object with `id`, `name`, `status` |
| get_plan | Returns object with `tasks` array |
| list_plans | Returns `{"count": N, "plans": [...]}` |
| update_plan | Returns object with updated fields |
| delete_plan | Returns `{"deleted": true, "tasks_deleted": N}` |
| create_task | Returns object with `id`, `content`, `status` |
| get_task | Returns object with `plans` array |
| list_tasks | Returns `{"count": N, "tasks": [...]}` |
| update_task | Returns object with updated fields |
| delete_task | Returns `{"deleted": true, "id": "..."}` |
| get_related | Returns `{"count": N, "nodes": [...]}` |

## Report Format

For each tool tested, report:
- Tool name
- Input payload
- Response (full JSON)
- Validation: PASS/FAIL
- Notes (if any issues)

## Known Issues to Watch For

1. **search_memories**: Requires quoting strings with special characters (colons, etc.)
2. **delete_plan with no tasks**: Previously bugged (fixed) — verify it works
3. **Task creation**: Requires at least one valid plan_id