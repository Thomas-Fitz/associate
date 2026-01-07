# Associate - Terminal AI Agent with Graph Memory

Associate is a terminal-based AI agent that wraps GitHub Copilot and enhances it with a persistent graph-based memory system using Neo4j. It maintains architectural knowledge, code patterns, and dependencies in a graph database for intelligent, context-aware assistance.

## Features

- ✅ **Docker-Managed Neo4j**: Automatically creates and manages a Neo4j container
- ✅ **Global Configuration**: Configure once, use from any directory
- ✅ **Repository Initialization**: Register repositories in the graph database
- ✅ **Memory Management**: Save and search contextual memories
- ✅ **Learning System**: Save and retrieve architectural patterns per repository
- ✅ **MCP Integration**: Model Context Protocol server for AI agent integration
- ✅ **Context Isolation**: Strict separation of memory between repositories
- ✅ **CLI Commands**: Easy-to-use commands for memory operations
- 🚧 **Code Scanning**: Automatic code structure discovery (planned)

## Requirements

- **Go 1.24+** (tested with Go 1.25.5)
- **Docker** (for Neo4j container management)

## Installation

```bash
# Clone the repository
git clone https://github.com/fitz/associate.git
cd associate

# Build the application
go build -o associate

# Install to PATH for global access
sudo mv associate /usr/local/bin/

# Verify installation
associate --help
```

## Quick Start

### 1. Configure Neo4j Password (Global Configuration)

The global configuration is stored in `~/.associate/config` and is used by all repositories.

```bash
# Set the Neo4j password globally (required)
associate config set --global NEO4J_PASSWORD yourpassword

# Optionally configure other global settings
associate config set --global NEO4J_URI neo4j://localhost:7687
associate config set --global NEO4J_USERNAME neo4j
associate config set --global NEO4J_DATABASE neo4j
```

You can also set repository-specific config (without `--global` flag) in a local `.env` file.

### 2. Initialize a Repository

Navigate to any repository and initialize it:

```bash
cd /path/to/your/repo
associate init

# Or initialize from anywhere
associate init /path/to/your/repo
```

**What happens:**
- Neo4j container is automatically created and started (if not running)
- Repository is registered in the graph database
- Language detection is performed
- A `Repo` node is created in Neo4j with isolation from other repos

### 3. Save and Search Memories

```bash
# Save a memory/note about the codebase
associate save-memory "Authentication uses JWT with 15min expiry" \
  --type architectural_decision \
  --tags auth,security

# Search for memories
associate search-memory "authentication"
associate search-memory --type architectural_decision
associate search-memory --tags auth --limit 5
```

### 4. Work with Multiple Repositories

Since Associate uses global configuration, you can work with multiple repositories easily:

```bash
# Initialize multiple repos
associate init ~/projects/frontend
associate init ~/projects/backend
associate init ~/projects/mobile

# Each repo has isolated memory
cd ~/projects/frontend
associate save-memory "Uses React 18 with TypeScript" --type stack

cd ~/projects/backend
associate save-memory "Uses Go with Gin framework" --type stack

# Memories are isolated - backend won't see frontend memories
```

## Commands

### Configuration

```bash
# Set a global configuration value (stored in ~/.associate/config)
associate config set --global KEY VALUE

# Set a local configuration value (stored in ./.env)
associate config set KEY VALUE

# Get a configuration value  
associate config get KEY

# List all configuration
associate config list
```

### Repository Management

```bash
# Initialize a repository
associate init [path]

# Refresh repository memory (placeholder)
associate refresh-memory [path]

# Reset repository memory (with confirmation)
associate reset-memory [path]
```

### Memory Operations

```bash
# Save a memory
associate save-memory "content" \
  --type <context_type> \
  --tags tag1,tag2 \
  --related-to <file_path>

# Search memories
associate search-memory [query] \
  --type <context_type> \
  --tags tag1,tag2 \
  --limit 10

# Context types: architectural_decision, bug_fix, performance, note, etc.
```

### MCP Integration

The MCP (Model Context Protocol) server allows AI agents to interact with the memory system:

```bash
# Start MCP server (used by AI agents)
associate mcp [path]
```

**Available MCP Tools:**
- `save_memory` - Save contextual memories
- `search_memory` - Search for relevant memories
- `save_learning` - Save architectural patterns
- `search_learnings` - Find architectural patterns
- `get_repo_context` - Get AGENTS.md if present

AI agents can use these tools to:
1. Save learnings as they work on the codebase
2. Search for relevant context before making changes
3. Access repository-specific instructions (AGENTS.md)
4. Build up architectural knowledge over time

## Configuration

Configuration follows a **three-tier hierarchy**:

1. **Local** (`.env` in current directory) - highest priority
2. **Global** (`~/.associate/config`) - fallback
3. **Environment variables** - final fallback
4. **Defaults** - used if nothing else is set

| Key | Default | Description |
|-----|---------|-------------|
| `NEO4J_URI` | `neo4j://localhost:7687` | Neo4j connection URI |
| `NEO4J_USERNAME` | `neo4j` | Neo4j username |
| `NEO4J_PASSWORD` | **(required)** | Neo4j password |
| `NEO4J_DATABASE` | `neo4j` | Neo4j database name |
| `NEO4J_IMAGE` | `neo4j:5.25-community` | Docker image to use |
| `NEO4J_CONTAINER_NAME` | `associate-neo4j` | Docker container name |
| `GITHUB_COPILOT_TOKEN` | *(optional)* | GitHub Copilot API token |

### Security

- Global config is stored in `~/.associate/config` (not committed to repos)
- Local `.env` files are automatically added to `.gitignore`
- Passwords are masked in CLI output
- Credentials are stored locally and never transmitted except to Neo4j

## Architecture

### Directory Structure

```
.
├── cmd/                    # Cobra command definitions
│   ├── root.go            # Root command with Docker initialization
│   ├── config.go          # Configuration commands
│   ├── init.go            # Repository initialization
│   ├── memory.go          # Memory refresh/reset commands
│   ├── memory_commands.go # Save/search memory commands
│   └── mcp.go             # MCP server command
├── internal/
│   ├── config/            # Configuration management
│   │   ├── config.go      # Local config with global fallback
│   │   ├── global.go      # Global config management
│   │   └── config_test.go
│   ├── docker/            # Docker container lifecycle
│   │   ├── docker.go
│   │   └── docker_test.go
│   ├── graph/             # Neo4j graph operations
│   │   ├── graph.go
│   │   └── graph_test.go
│   ├── mcp/               # Model Context Protocol server
│   │   └── server.go
│   └── scanner/           # Code structure scanning (planned)
├── .env                   # Local configuration (gitignored)
├── .env.example           # Example configuration
├── main.go                # Application entry point
└── go.mod                 # Go module definition
```

### Graph Schema

**Nodes:**
- `Repo`: Repository node with path, name, language
- `Code`: Code element (function, class, struct, etc.)
- `Memory`: Contextual memory or note saved by AI/user
- `Learning`: Architectural pattern or learning specific to a repo

**Relationships:**
- `(Repo)-[:CONTAINS]->(Code)`: Repository contains code elements
- `(Repo)-[:HAS_MEMORY]->(Memory)`: Repository has memories
- `(Repo)-[:HAS_LEARNING]->(Learning)`: Repository has learnings

**Properties:**
- Repo: `path`, `name`, `description`, `language`, `updated_at`
- Code: `type`, `name`, `file_path`, `description`, `signature`, `line_start`, `line_end`, `updated_at`
- Memory: `content`, `context_type`, `tags`, `related_to`, `updated_at`
- Learning: `pattern`, `category`, `description`, `examples`, `updated_at`

**Isolation:**
All queries include `repo_path` to ensure strict isolation between repositories. Each repository's knowledge is completely separate.

### Docker Management

The application automatically manages a Neo4j Docker container:

1. **First Run**: Creates container with configured credentials
2. **Subsequent Runs**: Starts container if stopped
3. **Health Checks**: Waits for Neo4j to be ready before proceeding

**Manual Docker Management** (if needed):

```bash
# View container status
docker ps -a | grep associate-neo4j

# View logs
docker logs associate-neo4j

# Stop container
docker stop associate-neo4j

# Remove container (will be recreated on next run)
docker rm associate-neo4j
```

## Development

### Running Tests

```bash
# Run all tests
go test ./...

# Run tests with verbose output
go test ./... -v

# Run tests for a specific package
go test ./internal/config/... -v
go test ./internal/docker/... -v
go test ./internal/graph/... -v
```

### Test Coverage

```bash
# Generate coverage report
go test ./... -coverprofile=coverage.out

# View coverage in browser
go tool cover -html=coverage.out
```

### Design Philosophy

This project follows a **Test-Driven Development (TDD)** approach:

1. **Red Phase**: Write failing tests first
2. **Green Phase**: Implement minimal code to pass tests
3. **Refactor Phase**: Improve code quality while keeping tests green

All packages have comprehensive test coverage with unit and integration tests.

## Troubleshooting

### Docker is not available

**Error:** `Docker is not available. Please install Docker and ensure it is running`

**Solution:** 
1. Install Docker Desktop or Docker Engine
2. Start Docker
3. Verify: `docker version`

### Failed to connect to Neo4j

**Error:** `failed to connect to Neo4j`

**Solution:**
1. Check if container is running: `docker ps | grep associate-neo4j`
2. View container logs: `docker logs associate-neo4j`
3. Verify password is correct: `./associate config get NEO4J_PASSWORD`
4. Try restarting container: `docker restart associate-neo4j`

### Repository not initialized

**Error:** `repository not initialized`

**Solution:**
```bash
associate init
```

### Global config not found

**Error:** `missing required configuration fields: NEO4J_PASSWORD`

**Solution:**
```bash
# Set global configuration
associate config set --global NEO4J_PASSWORD yourpassword
```

## Use Cases

### For AI Agents (MCP Integration)

AI agents can use the MCP server to build context over time:

```bash
# Start MCP server (typically called by AI tools)
associate mcp

# AI agent can then:
# 1. Save memories as it learns about the codebase
# 2. Search for relevant context before making changes
# 3. Save architectural patterns it discovers
# 4. Access repo-specific instructions (AGENTS.md)
```

### For Developers (CLI)

Developers can manually save and retrieve important context:

```bash
# Save architectural decisions
associate save-memory "We decided to use event sourcing for audit trail" \
  --type architectural_decision \
  --tags architecture,audit

# Document performance issues
associate save-memory "Database queries slow on user table - needs indexing" \
  --type performance \
  --tags database,optimization

# Search before making changes
associate search-memory "authentication"
associate search-memory --type performance --limit 5
```

### Multi-Repository Workflows

Work across multiple projects with isolated memories:

```bash
# Set up global config once
associate config set --global NEO4J_PASSWORD mypassword

# Initialize all your projects
associate init ~/work/frontend
associate init ~/work/backend
associate init ~/work/mobile
associate init ~/personal/side-project

# Each project maintains its own isolated knowledge
cd ~/work/backend
associate save-memory "Uses Go modules with vendor directory"

cd ~/work/frontend  
associate search-memory "Go modules"  # Won't find backend memory
```

## Roadmap

### Completed ✅
- Global configuration system
- Repository initialization and isolation
- Memory save/search operations
- Learning save/search operations
- MCP server for AI agent integration
- CLI commands for manual operations
- Comprehensive test coverage

### Next Steps 🚀
- [ ] Code scanning implementation
- [ ] Advanced search with vector similarity
- [ ] Visualization of graph relationships
- [ ] Multi-language code parser
- [ ] GitHub Copilot direct integration
- [ ] Web UI for graph visualization

## Contributing

Contributions are welcome! Please:

1. Fork the repository
2. Create a feature branch
3. Write tests for new functionality
4. Ensure all tests pass: `go test ./...`
5. Submit a pull request

## License
