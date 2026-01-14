#!/bin/bash
# MCP stdio wrapper for containerized associate (OPTIONAL)
#
# This script is a convenience wrapper for environments where you want
# automatic Neo4j startup. For most use cases, you can configure your
# MCP client to use docker directly:
#
#   "command": "docker",
#   "args": ["run", "-i", "--rm", "--network", "associate_default",
#            "-e", "NEO4J_URI=bolt://neo4j:7687", "associate-associate"]
#
# Use this script as the "command" in MCP client configuration when you don't have Go installed.
# It runs the associate MCP server via Docker, communicating over stdin/stdout.

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

cd "$PROJECT_ROOT"

# Ensure data directories exist
mkdir -p .associate/data .associate/logs

# Check if Neo4j is running, start if not
if ! docker ps --format '{{.Names}}' 2>/dev/null | grep -q '^associate-neo4j$'; then
    # Start Neo4j in the background
    docker-compose up -d neo4j >&2
    
    # Wait for Neo4j to be healthy (max 60 seconds)
    for i in {1..30}; do
        if docker exec associate-neo4j wget -q -O - http://localhost:7474 >/dev/null 2>&1; then
            break
        fi
        sleep 2
    done
fi

# Run associate in stdio mode, connecting to the neo4j container network
# --rm: Remove container after exit
# -T: Don't allocate a pseudo-TTY (important for stdio mode)
# --entrypoint: Override the default -http entrypoint to run in stdio mode
# Logs go to stderr, MCP protocol goes to stdin/stdout
exec docker-compose run --rm -T --entrypoint /associate associate
