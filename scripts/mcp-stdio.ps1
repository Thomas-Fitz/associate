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

$ErrorActionPreference = "Stop"

$SCRIPT_DIR = Split-Path -Parent $MyInvocation.MyCommand.Path
$PROJECT_ROOT = Split-Path -Parent $SCRIPT_DIR

Set-Location $PROJECT_ROOT

# Check if Neo4j is running, start if not
$neo4jRunning = docker ps --format "{{.Names}}" 2>$null | Select-String -Pattern "^associate-neo4j$" -Quiet

if (-not $neo4jRunning) {
    # Start Neo4j in the background
    docker-compose up -d neo4j 2>&1 | Out-Host
    
    # Wait for Neo4j to be healthy (max 60 seconds)
    $maxAttempts = 30
    for ($i = 1; $i -le $maxAttempts; $i++) {
        try {
            docker exec associate-neo4j wget -q -O - http://localhost:7474 2>&1 | Out-Null
            break
        }
        catch {
            Start-Sleep -Seconds 2
        }
    }
}

# Run associate in stdio mode, connecting to the neo4j container network
# --rm: Remove container after exit
# -T: Don't allocate a pseudo-TTY (important for stdio mode)
# --entrypoint: Override the default -http entrypoint to run in stdio mode
# Logs go to stderr, MCP protocol goes to stdin/stdout
docker-compose run --rm -T --entrypoint /associate associate
