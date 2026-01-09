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

### Pattern 3: Build Project Hierarchies

Use the PART_OF relationship to organize memories into logical project structures.

**User Prompt:**
> "I'm starting work on the payment processing module. Create a project memory for 'Payment System' and note that I'm implementing Stripe integration as part of it."

**Expected Agent Behavior:**
1. Create a Project-type memory for the overall payment system
2. Create a Note-type memory for Stripe integration
3. Link the implementation note to the project using PART_OF

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


### Pattern 6: Track Tasks with Dependencies

Use Task-type memories to track work items and their relationships to code and other tasks.

**User Prompt:**
> "Create a task to add unit tests for the payment webhook handler, which depends on the Stripe integration being completed."

**Expected Agent Behavior:**
1. Search for existing Stripe integration memory
2. Create Task-type memory
3. Link to related implementation using DEPENDS_ON


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

1. **Always search first**: Before creating new memories, search to prevent duplicates and discover related context
2. **Use specific tags**: Create a consistent tag taxonomy (e.g., "architecture", "bug-fix", "performance")
3. **Rich metadata**: Include file paths, dates, version numbers, and other contextual data
4. **Meaningful relationships**: Use PART_OF for hierarchies, DEPENDS_ON for dependencies, REFERENCES for citations, RELATES_TO for general connections, BLOCKS for task dependencies, FOLLOWS for sequences, IMPLEMENTS for code-to-decision links
5. **Type appropriately**: Use Note for observations, Task for work items, Project for initiatives, Repository for codebase-wide info
6. **Update, don't duplicate**: When you gain new information about existing concepts, update the memory rather than creating a new one
7. **Cross-reference decisions**: Always link architectural decisions to the code they affect
8. **Progressive refinement**: Start with basic memories and enhance them with relationships as you learn more about the codebase
9. **Use get_related for exploration**: When you have an ID, use `get_related` to discover connected context without searching
10. **Clean up with delete_memory**: Remove outdated or incorrect memories to keep the knowledge base accurate

### Pattern 9: Retrieve and Explore from ID

When you have a memory ID (from a previous search or stored in context), retrieve its full details and explore its connections.

**User Prompt:**
> "Get the details of that authentication task I created earlier and show me what it depends on."

**Expected Agent Behavior:**
1. Use `get_memory` to retrieve the full memory
2. Use `get_related` to explore its dependencies

**Example Tool Calls:**
```json
// First, get the memory details
{
  "name": "get_memory",
  "arguments": {
    "id": "auth-task-uuid"
  }
}

// Then, explore what it depends on
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

### Pattern 10: Build Task Dependency Chains

Use BLOCKS and FOLLOWS relationships to create ordered task lists that agents can traverse.

**User Prompt:**
> "Create a task list for adding user authentication. It should have: 1) Design auth schema, 2) Implement user model, 3) Add JWT tokens, 4) Create login endpoint."

**Expected Agent Behavior:**
1. Create each task as a Task-type memory
2. Link them with FOLLOWS relationships to establish sequence
3. Use BLOCKS to indicate which tasks gate others

**Example Tool Calls:**
```json
// Create the first task
{
  "name": "add_memory",
  "arguments": {
    "content": "Design authentication database schema including users, sessions, and tokens tables",
    "type": "Task",
    "metadata": {"status": "pending", "priority": "1"},
    "tags": ["auth", "database", "design"]
  }
}
// Returns: {"id": "task-1-uuid", ...}

// Create second task that follows the first
{
  "name": "add_memory",
  "arguments": {
    "content": "Implement User model with password hashing and validation",
    "type": "Task",
    "metadata": {"status": "pending", "priority": "2"},
    "tags": ["auth", "model", "implementation"],
    "follows": ["task-1-uuid"]
  }
}
// Returns: {"id": "task-2-uuid", ...}

// Create third task
{
  "name": "add_memory",
  "arguments": {
    "content": "Add JWT token generation and validation",
    "type": "Task",
    "metadata": {"status": "pending", "priority": "3"},
    "tags": ["auth", "jwt", "implementation"],
    "follows": ["task-2-uuid"]
  }
}
// Returns: {"id": "task-3-uuid", ...}

// Create final task that depends on JWT
{
  "name": "add_memory",
  "arguments": {
    "content": "Create login endpoint with JWT response",
    "type": "Task",
    "metadata": {"status": "pending", "priority": "4"},
    "tags": ["auth", "api", "endpoint"],
    "follows": ["task-3-uuid"],
    "depends_on": ["task-3-uuid"]
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

### Pattern 12: Clean Up Completed or Invalid Memories

Remove memories that are no longer relevant.

**User Prompt:**
> "The authentication feature is complete. Clean up the task memories."

**Expected Agent Behavior:**
1. Search for related task memories
2. Delete each completed task

**Example Tool Calls:**
```json
// Search for auth tasks
{
  "name": "search_memories",
  "arguments": {
    "query": "authentication task",
    "limit": 10
  }
}

// Delete each completed task
{
  "name": "delete_memory",
  "arguments": {
    "id": "task-1-uuid"
  }
}
```
