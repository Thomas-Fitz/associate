# Associate - Usage Examples

## Getting Started

### 1. First Time Setup

```bash
# Build the application
go build -o associate

# Set your Neo4j password (required)
./associate config set NEO4J_PASSWORD mySecurePassword123

# Verify configuration
./associate config list
```

**Output:**
```
Configuration:
  NEO4J_URI: neo4j://localhost:7687
  NEO4J_USERNAME: neo4j
  NEO4J_PASSWORD: my****23
  NEO4J_DATABASE: neo4j
  NEO4J_IMAGE: neo4j:5.25-community
  NEO4J_CONTAINER_NAME: associate-neo4j
```

### 2. Initialize a Repository

```bash
# Initialize current directory
./associate init

# Or initialize a specific project
./associate init ~/projects/my-go-app
```

**First Run Output:**
```
✓ Created Neo4j container 'associate-neo4j'
  Waiting for Neo4j to be ready...
  ✓ Neo4j is ready
✓ Initialized repository 'my-go-app'
  Path: /Users/you/projects/my-go-app
  Language: Go

Repository is now registered in the graph database.
Use 'associate refresh-memory' to scan and index the codebase.
```

**Subsequent Runs:**
- If container is stopped, it starts automatically
- If container is running, command proceeds immediately

## Working with Configuration

### View All Settings

```bash
./associate config list
```

### Update Specific Settings

```bash
# Change Neo4j URI
./associate config set NEO4J_URI neo4j://192.168.1.100:7687

# Change container name
./associate config set NEO4J_CONTAINER_NAME my-neo4j

# Set GitHub Copilot token (for future use)
```

### Retrieve a Single Value

```bash
./associate config get NEO4J_PASSWORD
# Output: NEO4J_PASSWORD=mySecurePassword123

./associate config get NEO4J_URI
# Output: NEO4J_URI=neo4j://localhost:7687
```

## Repository Memory Management

### Initialize Multiple Repositories

```bash
# Initialize three different projects
./associate init ~/projects/frontend-app
./associate init ~/projects/backend-api
./associate init ~/projects/data-pipeline
```

Each repository gets its own isolated graph in Neo4j. Memory from one repo **never** mixes with another.

### Refresh Repository Memory

```bash
# Refresh current directory
./associate refresh-memory

# Refresh specific repository
./associate refresh-memory ~/projects/my-go-app
```

**Output:**
```
Refreshing memory for: /Users/you/projects/my-go-app
Scanning codebase...
✓ Memory refresh complete

Note: Full code scanning will be implemented in the next phase.
```

### Reset Repository Memory

```bash
./associate reset-memory
```

**Interactive Prompt:**
```
⚠️  WARNING: This will permanently delete all memory for repository 'my-go-app'
   Path: /Users/you/projects/my-go-app

Are you sure you want to continue? (yes/no): yes

Deleting memory for 'my-go-app'...
✓ Memory reset complete

Run 'associate init' to re-initialize the repository.
```

## Docker Management

### Automatic Container Management

The Neo4j container is managed automatically:

- **Created** on first run if it doesn't exist
- **Started** automatically if stopped
- **Health checked** before proceeding with commands

### Manual Docker Operations

```bash
# Check container status
docker ps | grep associate-neo4j

# View container logs
docker logs associate-neo4j

# View real-time logs
docker logs -f associate-neo4j

# Restart container
docker restart associate-neo4j

# Stop container (will auto-start on next associate command)
docker stop associate-neo4j

# Remove container (will be recreated on next run)
docker stop associate-neo4j
docker rm associate-neo4j
```

## Accessing Neo4j Browser

Once the container is running, you can access the Neo4j Browser:

1. Open browser to: http://localhost:7474
2. Connect with:
   - URL: `neo4j://localhost:7687`
   - Username: `neo4j`
   - Password: (your configured password)

### Example Queries

```cypher
// View all repositories
MATCH (r:Repo) RETURN r

// View specific repository with its code nodes
MATCH (r:Repo {name: 'my-go-app'})-[:CONTAINS]->(c:Code)
RETURN r, c

// Count code elements by type
MATCH (r:Repo {name: 'my-go-app'})-[:CONTAINS]->(c:Code)
RETURN c.type, count(c) as count
ORDER BY count DESC

// Find all functions in a specific file
MATCH (c:Code {type: 'function', file_path: 'main.go'})
RETURN c.name, c.description, c.signature
```

## Common Workflows

### Workflow 1: New Project Setup

```bash
# 1. Configure once
./associate config set NEO4J_PASSWORD myPassword

# 2. Initialize your project
cd ~/projects/new-project
./associate init

# 3. Scan the codebase (when implemented)
./associate refresh-memory
```

### Workflow 2: Working with Multiple Projects

```bash
# Switch between projects - init each once
cd ~/projects/project-a
./associate init

cd ~/projects/project-b
./associate init

# Later, refresh any project
cd ~/projects/project-a
./associate refresh-memory

# Memories are completely isolated
```

### Workflow 3: Clean Slate

```bash
# Reset a project's memory
cd ~/projects/my-project
./associate reset-memory
# Confirm: yes

# Re-initialize
./associate init

# Scan fresh
./associate refresh-memory
```

## Troubleshooting Examples

### Problem: Docker not running

```bash
./associate init
```

**Error:**
```
Error: failed to ensure Neo4j container: Docker is not available. 
Please install Docker and ensure it is running
```

**Solution:**
1. Start Docker Desktop
2. Verify: `docker version`
3. Try again: `./associate init`

### Problem: Wrong password

```bash
./associate init
```

**Error:**
```
Error: failed to connect to Neo4j: authentication failed
```

**Solution:**
```bash
# Check current password
./associate config get NEO4J_PASSWORD

# Update if wrong
./associate config set NEO4J_PASSWORD correctPassword

# Or reset container
docker stop associate-neo4j
docker rm associate-neo4j
./associate init  # Creates with new password
```

### Problem: Repository not found

```bash
./associate refresh-memory
```

**Error:**
```
Error: repository not initialized. Run 'associate init' first
```

**Solution:**
```bash
./associate init
```

## Advanced Usage

### Using Different Neo4j Instances

```bash
# Development environment
./associate config set NEO4J_URI neo4j://localhost:7687
./associate config set NEO4J_CONTAINER_NAME associate-neo4j-dev

# Production environment (different config file)
cd /path/to/prod
./associate config set NEO4J_URI neo4j://prod-server:7687
./associate config set NEO4J_CONTAINER_NAME associate-neo4j-prod
```

### Working Directory Override

```bash
# Process a different directory without changing current directory
./associate init --dir ~/projects/other-project
./associate refresh-memory --dir ~/projects/other-project
```

## Testing Your Setup

```bash
# 1. Run tests
go test ./... -v

# 2. Test config
./associate config set TEST_KEY test_value
./associate config get TEST_KEY
./associate config list

# 3. Test init in temp directory
mkdir -p /tmp/test-repo
echo "package main" > /tmp/test-repo/main.go
./associate init /tmp/test-repo

# 4. Verify in Neo4j Browser
# Open http://localhost:7474
# Run: MATCH (r:Repo) RETURN r
```
