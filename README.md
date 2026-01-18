# Associate

A containerized Go application that provides AI coding agents with a Neo4j graph database "memory" through the Model Context Protocol (MCP).

## Overview

Associate allows AI agents to store, search, and connect information as they work on coding tasks. Think of it as a persistent memory layer for your AI assistant so you don't have to manage so many markdown files.

**Key Features:**
- **MCP Protocol**: Standard protocol for AI agent communication
- **Flexible**: stdio or HTTP transport modes
- **Containerized**: Docker-based deployment for easy setup
- **Graph Database**: Neo4j for storing memories with relationships

## Quick Start

### MCP Client Configuration

This project implements an MCP server that can be used by local agents (Copilot, Copilot CLI, Claude Desktop, Cursor, etc.). Two transport modes are supported:

| Transport | Use Case | How to Configure |
|-----------|----------|------------------|
| **stdio** | IDE integrations (VS Code, Cursor, Claude Desktop) | Direct `docker run` command |
| **HTTP** | Copilot CLI, remote access, multi-client | `docker-compose up -d` |

#### Option A - stdio Transport (Recommended for IDEs)

The simplest approach uses `docker run` directly. First, ensure the DB container is running:

```bash
docker-compose up -d neo4j
```

Then configure your MCP client to launch Associate via Docker:

**VS Code / Cursor / Claude Desktop:**

```json
{
  "servers": {
    "associate": {
      "command": "docker",
      "args": [
        "run", "-i", "--rm",
        "--network", "associate_default",
        "-e", "NEO4J_URI=bolt://neo4j:7687",
        "associate-associate"
      ]
    }
  }
}
```

This configuration:
- Uses `-i` for interactive stdin/stdout (required for stdio transport)
- Uses `--rm` for automatic cleanup
- Connects to the DB container via Docker network
- Waits for DB automatically (built-in retry logic)

**Alternative: Using Helper Scripts**

For convenience, wrapper scripts to have your IDE start the DB container are provided in `scripts/`:

Unix:
```json
{
  "mcpServers": {
    "associate": {
      "command": "/path/to/associate/scripts/mcp-stdio.sh"
    }
  }
}
```

Windows PowerShell:
```json
{
  "servers": {
    "associate": {
      "type": "stdio",
      "command": "powershell",
      "args": ["-ExecutionPolicy", "Bypass", "-File"  "\\path\\to\\associate\\scripts\\mcp-stdio.ps1"]
    }
  }
}
```

#### Option B - HTTP Transport

For HTTP mode, start the full stack with docker-compose:

```bash
docker-compose up -d
```

Then configure your MCP client to connect via HTTP:

```json
{
  "mcpServers": {
    "associate": {
      "type": "http",
      "url": "http://localhost:8080"
    }
  }
}
```

**Note:** Copilot CLI works best with HTTP transport.

### Prompting & AGENTS.md

Associate tools can be triggered manually through prompts or by updating your AGENTS.md. See the [prompt documentation](docs/prompts.md) for examples.

```
# snippets from successful AGENTS.md use cases
"When planning, check your memory for context related to the current task."

"Before creating new memories, always search to check if similar information already exists."

"Use Plans to organize multi-step work and Tasks to track actionable items with status."

"As you learn, update existing memories with new relationships and information."

"Use create_task with plan_ids (required array) to associate tasks with plans for organized tracking."

"Update task status as you work: pending → in_progress → completed."
```
### Memory DB Access

Once the DB container is running, access the Neo4j UI in your browser at http://localhost:7474.

### Data Persistence

Neo4j data is stored in a Docker named volume that persists across container restarts:

- `associate_neo4j_data` - Database files

**Volume Management:**

```bash
# List volumes
docker volume ls | grep associate

# Inspect volume details
docker volume inspect associate_neo4j_data

# Remove all data (stops containers and deletes volumes)
docker-compose down -v

# Stop containers but keep data (for restarts)
docker-compose down
```

**Data Lifecycle:**
- `docker-compose down` - Stops containers, **preserves data**
- `docker-compose down -v` - Stops containers and **removes all data**
- Volumes persist even if you delete the containers manually

**Backup/Migration:**
To backup or migrate your Neo4j data, use Docker volume commands or Neo4j's native backup tools.


## MCP Tools

### Memory Tools

| Function | Description |
| :--- | :--- |
| `search_memories` | Search for memories by content with full-text search. |
| `add_memory` | Create a new memory with optional relationships. |
| `update_memory` | Update an existing memory or add new relationships. |
| `get_memory` | Retrieve a single memory by ID, including its relationships. |
| `delete_memory` | Delete a memory and all its relationships from the graph. |
| `get_related` | Traverse the graph to find all nodes (Memory, Plan, Task) connected to a given node. Supports filtering by relationship type, direction, and traversal depth. |

### Plan Tools

| Function | Description |
| :--- | :--- |
| `create_plan` | Create a new plan for organizing related tasks. |
| `get_plan` | Retrieve a plan by ID, including its tasks. |
| `update_plan` | Update a plan's name, description, status, or relationships. |
| `delete_plan` | Delete a plan and cascade delete orphan tasks. |
| `list_plans` | List all plans, optionally filtered by status or tags. |

### Task Tools

| Function | Description |
| :--- | :--- |
| `create_task` | Create a new task, optionally linked to a plan. |
| `get_task` | Retrieve a task by ID, including its plans and relationships. |
| `update_task` | Update a task's content, status, or relationships. |
| `delete_task` | Delete a task from the graph. |
| `list_tasks` | List tasks, optionally filtered by plan, status, or tags. |

## Node Types

Associate uses three distinct node types in the graph:

### Memories
General knowledge storage for notes, documentation, and context.

**Memory Types:**
- `Note` - General notes and observations
- `Repository` - Code repository information  
- `Memory` - Generic memories (default)

### Plans
Containers for organizing related tasks with status tracking.

**Plan Statuses:**
- `draft` - Plan is being defined
- `active` - Plan is currently being worked on
- `completed` - All tasks in the plan are done
- `archived` - Plan is no longer active

### Tasks
Actionable work items with status tracking and dependencies.

**Task Statuses:**
- `pending` - Task has not started
- `in_progress` - Task is actively being worked on
- `completed` - Task is finished
- `cancelled` - Task was cancelled
- `blocked` - Task is blocked by dependencies

## Relationship Types

- `RELATES_TO` - General relationship
- `PART_OF` - Hierarchical containment
- `REFERENCES` - Reference/citation
- `DEPENDS_ON` - Dependency relationship
- `BLOCKS` - Task blocking relationship (A blocks B means A must complete before B can start)
- `FOLLOWS` - Sequence ordering (A follows B in a workflow)
- `IMPLEMENTS` - Implementation relationship (code implements a decision/task)

## Configuration

Environment variables:

| Variable | Default | Description |
|----------|---------|-------------|
| `NEO4J_URI` | `bolt://localhost:7687` | Neo4j connection URI |
| `NEO4J_USERNAME` | `neo4j` | Neo4j username |
| `NEO4J_PASSWORD` | `password` | Neo4j password |
| `NEO4J_DATABASE` | `neo4j` | Neo4j database name |

## Roadmap

* ~~Better To Do list handling including "get to do by project"~~ ✅ (Implemented as Plans/Tasks with `list_tasks` by plan_id)
* Improved search result weights
* Search across all node types (currently Memory-only)
* Deep storage for long term memory
* Distinct agent memory databases
* GUI

## Troubleshooting

TODO

## License

See [LICENSE](LICENSE) for details.
