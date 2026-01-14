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

### Docker (Recommended)

```bash
# Start Neo4j and the Associate server
docker-compose up -d

# The MCP server will be available at http://localhost:8080

# The Neo4j server will be available at http://localhost:7474
```

### MCP Client Configuration Options

This project implements an MCP server that can be used by local agents (Copilot, Copilot CLI, Claude, etc.). Two common transport modes are supported:

- Stdio/Command (recommended): the MCP server can be run as a command that a client launches and communicates with over stdin/stdout
- HTTP: The MCP server listens at http://localhost:8080


#### Option A - IDE Integration with stdio (Recommended)

1. Start the Associate server from the repo

```bash
docker compose up -d
```

2. In VS Code, add an MCP server entry that points at the correct mcp-stdio scrpit (.sh for Unix, .ps1 for Windows).

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

Powershell (Windows):

```json
{
	"servers": {
		"associate": {
			"type": "stdio",
			"command": "powershell",
			"args": ["-ExecutionPolicy","Bypass","-File","/path/to/associate/scripts/mcp-stdio/mcp-stdio.ps1"]
		}
	},
	"inputs": []
}
```

The script automatically:
- Starts Neo4j if not already running
- Runs the associate container in stdio mode
- Routes MCP protocol messages through stdin/stdout

Notes and best practices for Copilot:
- Use the Copilot UI (Extensions / Commands) to manage and trust local MCP servers.
- Ensure Agent/Chat features are enabled in Copilot settings so the agent can call tools exposed by Associate. To do so, add associate to your allows tools in your AGENTS.md.

**Requirements:** Docker and docker-compose only.

#### Option B - Github Copilot CLI

Support for stdio is limited in Copilot CLI as of 0.0.377. Use HTTP if you run into problems.

Example HTTP config:

```json
// ~/.copilot/mcp-config.json
{
  "mcpServers": {
    "associate": {
      "type": "http",
      "url": "http://localhost:8080",
      "headers": {},
      "tools": [
        "*"
      ]
    }
  }
}
```

### Prompting & AGENTS.md

Associate tools can be triggered manually through prompts or by updating your AGENTS.md. See the [prompt documentation](docs\prompts.md) for examples.

```
# snippets from successful AGENTS.md use cases
"When planning, check if your memory context related to the current task."

"Before creating new memories, always search to check if similar information already exists."

"Use the PART_OF relationship to organize memories into logical project structures."

"As you learn, update existing memories with new relationships and information."

"Use Task-type memories to track work items and their relationships to code and other tasks."

Store repository-wide information and conventions using Repository-type memories."
```

## MCP Tools

| Function | Description |
| :--- | :--- |
| `search_memories` | Search for memories by content with full-text search. |
| `add_memory` | Create a new memory with optional relationships. |
| `update_memory` | Update an existing memory or add new relationships. |
| `get_memory` | Retrieve a single memory by ID, including its relationships. |
| `delete_memory` | Delete a memory and all its relationships from the graph. |
| `get_related` | Traverse the graph to find all memories connected to a given node. Supports filtering by relationship type, direction, and traversal depth. |

## Memory Types

- `Note` - General notes and observations
- `Task` - Tasks and action items
- `Project` - Project definitions
- `Repository` - Code repository information
- `Memory` - Generic memories (default)

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

* Better To Do list handling including "get to do by project"
* Improved search result weights
* Indexing
* Deep storage for long term memory
* Distinct agent memory databases
* GUI

## Troubleshooting

TODO

## License

See [LICENSE](LICENSE) for details.
