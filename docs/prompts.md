## Agent Usage Patterns

This section provides sample prompts and workflows to help AI coding agents effectively use Associate's memory system during development tasks. These patterns follow MCP best practices for context-aware operations and progressive memory building.

### Pattern 1: Search-First to Avoid Duplicates

Before creating new memories, always search to check if similar information already exists.

**User Prompt:**
> "I'm working on implementing authentication in the user service. Check if we have any existing notes about authentication patterns in this codebase."

**Expected Agent Behavior:**
1. Use `search_memories` with query: "authentication user service patterns"
2. Review results to find relevant existing memories
3. If found, reference existing memory IDs in new work
4. If not found, create new memory with rich context

### Pattern 2: Create Rich, Connected Memories

When saving new information, include comprehensive metadata, tags, and relationships to related concepts.

**User Prompt:**
> "I just implemented JWT token validation in internal/auth/jwt.go. Save this architectural decision along with the rationale that we chose JWT over session cookies for scalability."

**Expected Agent Behavior:**
1. Create a memory with type "Note"
2. Include file path in metadata
3. Add relevant tags (architecture, authentication, jwt)
4. Link to related memories if any exist

### Pattern 3: Build Project Hierarchies with Plans

Use the `create_plan` tool to organize work into logical structures with tasks.

**User Prompt:**
> "I'm starting work on the payment processing module. Create a plan for 'Payment System' and add a task for implementing Stripe integration."

**Expected Agent Behavior:**
1. Create a Plan for the overall payment system using `create_plan`
2. Create a Task for Stripe integration using `create_task` with `plan_ids` (required)

**Example Tool Calls:**
```json
// Create the plan
{
  "name": "create_plan",
  "arguments": {
    "name": "Payment System",
    "description": "Implement payment processing module with Stripe integration",
    "status": "active",
    "tags": ["payments", "stripe", "backend"]
  }
}
// Returns: {"id": "plan-uuid", ...}

// Create task linked to the plan (plan_ids is required)
{
  "name": "create_task",
  "arguments": {
    "content": "Implement Stripe payment gateway integration",
    "plan_ids": ["plan-uuid"],
    "status": "pending",
    "tags": ["stripe", "integration"]
  }
}
```

### Pattern 4: Progressive Memory Enhancement

As you learn more context, update existing memories with new relationships and information.

**User Prompt:**
> "I discovered that the JWT implementation depends on the user repository for token validation. Update the JWT memory to reflect this dependency."

**Expected Agent Behavior:**
1. Search for the JWT memory
2. Search for user repository memory (or create if needed)
3. Update JWT memory to add DEPENDS_ON relationship


### Pattern 5: Cross-Reference Decisions and Code

Link architectural decisions to the code files they affect using REFERENCES relationships.

**User Prompt:**
> "Document the decision to use Redis for rate limiting, and link it to the middleware implementation in internal/middleware/ratelimit.go"

**Expected Agent Behavior:**
1. Create a Note for the architectural decision
2. Create or find a memory for the middleware file
3. Link decision to implementation using REFERENCES


### Pattern 6: Track Tasks with the Task Tools

Use the dedicated Task tools (`create_task`, `get_task`, `update_task`, `list_tasks`) to manage work items with proper status tracking.

**User Prompt:**
> "Create a task to add unit tests for the payment webhook handler, which depends on the Stripe integration being completed."

**Expected Agent Behavior:**
1. Search for existing Stripe integration task or memory
2. Create a task using `create_task` with `depends_on` relationship
3. Link to a plan if one exists

**Example Tool Calls:**
```json
// Search for related context
{
  "name": "search_memories",
  "arguments": {
    "query": "stripe integration payment",
    "limit": 5
  }
}

// Create the task with dependency (plan_ids is required)
{
  "name": "create_task",
  "arguments": {
    "content": "Add unit tests for payment webhook handler",
    "status": "blocked",
    "plan_ids": ["payment-plan-uuid"],
    "depends_on": ["stripe-task-uuid"],
    "tags": ["testing", "webhooks", "payments"]
  }
}
```


### Pattern 7: Repository-Level Context

Store repository-wide information and conventions using Repository-type memories.

**User Prompt:**
> "Document that this codebase uses the repository pattern for data access and all repositories should implement the Repository interface."

**Expected Agent Behavior:**
1. Create Repository-type memory for codebase conventions
2. Include examples and rationale
3. Tag for easy discovery

### Pattern 8: Multi-Turn Context Building

Agents should build context progressively across multiple interactions within a session.

**User Session Example:**

**Turn 1 - User:** "I'm starting to work on the notification system."

**Agent Action:** Search for existing notification-related memories
```json
{"name": "search_memories", "arguments": {"query": "notification system", "limit": 10}}
```

**Turn 2 - User:** "I'm implementing email notifications using SendGrid."

**Agent Action:** Create memory and search for any SendGrid configuration
```json
{"name": "add_memory", "arguments": {
  "content": "Implementing email notifications with SendGrid API...",
  "type": "Note",
  "tags": ["notifications", "email", "sendgrid"]
}}
```

**Turn 3 - User:** "This connects to the user preferences system to check if users want email notifications."

**Agent Action:** Update the notification memory with a DEPENDS_ON relationship
```json
{"name": "update_memory", "arguments": {
  "id": "<notification-memory-id>",
  "depends_on": ["<user-preferences-memory-id>"]
}}
```

### Best Practices for Agents Using Associate

1. **Always search first**: Before creating new memories, plans, or tasks, search to prevent duplicates and discover related context
2. **Use specific tags**: Create a consistent tag taxonomy (e.g., "architecture", "bug-fix", "performance")
3. **Rich metadata**: Include file paths, dates, version numbers, and other contextual data
4. **Meaningful relationships**: Use PART_OF for hierarchies, DEPENDS_ON for dependencies, REFERENCES for citations, RELATES_TO for general connections, BLOCKS for task dependencies, FOLLOWS for sequences, IMPLEMENTS for code-to-decision links
5. **Choose the right type**:
   - Use **Plans** (`create_plan`) to organize multi-step work with status tracking (draft → active → completed → archived)
   - Use **Tasks** (`create_task`) for actionable work items with status (pending → in_progress → completed/cancelled/blocked)
   - Use **Memories** (`add_memory`) for knowledge, notes, and documentation (Note, Repository types)
6. **Track task status**: Update task status as work progresses using `update_task` - don't just create and forget
7. **Link tasks to plans**: Always associate tasks with at least one plan using `plan_ids` (required) for better organization
8. **Update, don't duplicate**: When you gain new information about existing concepts, update the memory/task/plan rather than creating a new one
9. **Cross-reference decisions**: Link tasks and memories to architectural decisions using IMPLEMENTS or REFERENCES
10. **Progressive refinement**: Start with basic items and enhance them with relationships as you learn more about the codebase
11. **Use get_related for exploration**: When you have an ID, use `get_related` to discover connected context across all node types (Memory, Plan, Task)
12. **Complete plans properly**: When finishing work, update task statuses to "completed" and plan status to "completed"
13. **Use list_plans/list_tasks**: Monitor active work with `list_plans` and `list_tasks` with status filters

### Pattern 9: Retrieve and Explore from ID

When you have a memory, plan, or task ID (from a previous search or stored in context), retrieve its full details and explore its connections.

**User Prompt:**
> "Get the details of that authentication task I created earlier and show me what it depends on."

**Expected Agent Behavior:**
1. Use `get_task` to retrieve the task details (or `get_memory`/`get_plan` for other types)
2. Use `get_related` to explore its dependencies

**Example Tool Calls:**
```json
// Get the task details
{
  "name": "get_task",
  "arguments": {
    "id": "auth-task-uuid"
  }
}

// Explore what it depends on
{
  "name": "get_related",
  "arguments": {
    "id": "auth-task-uuid",
    "relationship_type": "DEPENDS_ON",
    "direction": "outgoing",
    "depth": 2
  }
}
```

### Pattern 10: Build Task Dependency Chains with Plans

Use Plans to organize related tasks and use task status tracking to manage progress.

**User Prompt:**
> "Create a task list for adding user authentication. It should have: 1) Design auth schema, 2) Implement user model, 3) Add JWT tokens, 4) Create login endpoint."

**Expected Agent Behavior:**
1. Create a Plan for the authentication feature
2. Create each task linked to the plan with FOLLOWS relationships for sequence
3. Use `list_tasks` with `plan_id` filter to track progress

**Example Tool Calls:**
```json
// Create the plan
{
  "name": "create_plan",
  "arguments": {
    "name": "User Authentication Implementation",
    "description": "Add user authentication with JWT tokens",
    "status": "active",
    "tags": ["auth", "feature"]
  }
}
// Returns: {"id": "auth-plan-uuid", ...}

// Create the first task (plan_ids is required)
{
  "name": "create_task",
  "arguments": {
    "content": "Design authentication database schema including users, sessions, and tokens tables",
    "plan_ids": ["auth-plan-uuid"],
    "status": "pending",
    "metadata": {"priority": "1"},
    "tags": ["auth", "database", "design"]
  }
}
// Returns: {"id": "task-1-uuid", ...}

// Create second task that follows the first
{
  "name": "create_task",
  "arguments": {
    "content": "Implement User model with password hashing and validation",
    "plan_ids": ["auth-plan-uuid"],
    "status": "pending",
    "metadata": {"priority": "2"},
    "tags": ["auth", "model", "implementation"],
    "follows": ["task-1-uuid"]
  }
}
// Returns: {"id": "task-2-uuid", ...}

// Create third task
{
  "name": "create_task",
  "arguments": {
    "content": "Add JWT token generation and validation",
    "plan_ids": ["auth-plan-uuid"],
    "status": "pending",
    "metadata": {"priority": "3"},
    "tags": ["auth", "jwt", "implementation"],
    "follows": ["task-2-uuid"]
  }
}
// Returns: {"id": "task-3-uuid", ...}

// Create final task
{
  "name": "create_task",
  "arguments": {
    "content": "Create login endpoint with JWT response",
    "plan_ids": ["auth-plan-uuid"],
    "status": "pending",
    "metadata": {"priority": "4"},
    "tags": ["auth", "api", "endpoint"],
    "follows": ["task-3-uuid"],
    "depends_on": ["task-3-uuid"]
  }
}

// List all tasks in the plan
{
  "name": "list_tasks",
  "arguments": {
    "plan_id": "auth-plan-uuid"
  }
}
```

### Pattern 11: Traverse the Dependency Graph

Use `get_related` to understand what a task blocks or what blocks it.

**User Prompt:**
> "What tasks are blocked by the database schema design task?"

**Expected Agent Behavior:**
1. Use `get_related` with direction "incoming" to find what depends on this task

**Example Tool Call:**
```json
{
  "name": "get_related",
  "arguments": {
    "id": "task-1-uuid",
    "relationship_type": "FOLLOWS",
    "direction": "incoming",
    "depth": 3
  }
}
```

### Pattern 12: Complete Plans and Clean Up Tasks

Use status updates to mark work as complete and optionally delete completed plans.

**User Prompt:**
> "The authentication feature is complete. Mark it done and archive the plan."

**Expected Agent Behavior:**
1. Update each task status to `completed` using `update_task`
2. Update the plan status to `completed` using `update_plan`
3. Optionally archive or delete the plan

**Example Tool Calls:**
```json
// List tasks in the plan to find incomplete ones
{
  "name": "list_tasks",
  "arguments": {
    "plan_id": "auth-plan-uuid",
    "status": "pending"
  }
}

// Update each task to completed
{
  "name": "update_task",
  "arguments": {
    "id": "task-1-uuid",
    "status": "completed"
  }
}

// Mark the plan as completed
{
  "name": "update_plan",
  "arguments": {
    "id": "auth-plan-uuid",
    "status": "completed"
  }
}

// Or delete the plan (will cascade delete orphan tasks)
{
  "name": "delete_plan",
  "arguments": {
    "id": "auth-plan-uuid"
  }
}
```

### Pattern 13: View All Active Plans

Use `list_plans` to see all ongoing work.

**User Prompt:**
> "What plans do I have in progress?"

**Expected Agent Behavior:**
1. Use `list_plans` with status filter for "active"

**Example Tool Call:**
```json
{
  "name": "list_plans",
  "arguments": {
    "status": "active"
  }
}
```

### Pattern 14: Get Plan Overview with Tasks

Use `get_plan` to retrieve a plan with its associated tasks.

**User Prompt:**
> "Show me the details of the Payment System plan and all its tasks."

**Expected Agent Behavior:**
1. Use `get_plan` to get the plan details and linked tasks

**Example Tool Call:**
```json
{
  "name": "get_plan",
  "arguments": {
    "id": "payment-plan-uuid"
  }
}
// Returns plan details with nested tasks array
```

### Pattern 15: Update Task Progress

Track work progress by updating task statuses.

**User Prompt:**
> "I'm starting work on the JWT token implementation."

**Expected Agent Behavior:**
1. Find the task using `list_tasks` or `search_memories`
2. Update the task status to `in_progress`

**Example Tool Calls:**
```json
// Find the task
{
  "name": "list_tasks",
  "arguments": {
    "plan_id": "auth-plan-uuid"
  }
}

// Update status to in_progress
{
  "name": "update_task",
  "arguments": {
    "id": "jwt-task-uuid",
    "status": "in_progress"
  }
}
```

### Pattern 16: Cross-Reference Tasks with Memories

Link tasks to relevant documentation and architectural decisions.

**User Prompt:**
> "Link the JWT task to our authentication architecture memory."

**Expected Agent Behavior:**
1. Use `update_task` to add a `references` relationship to the memory

**Example Tool Call:**
```json
{
  "name": "update_task",
  "arguments": {
    "id": "jwt-task-uuid",
    "references": ["auth-architecture-memory-uuid"]
  }
}
```

