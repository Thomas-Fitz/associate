# MCP stdio wrapper for containerized associate (OPTIONAL)
#
# This script is a convenience wrapper for environments where you want
# automatic PostgreSQL/AGE startup. For most use cases, you can configure your
# MCP client to use docker directly:
#
#   "command": "docker",
#   "args": ["run", "-i", "--rm", "--network", "associate_default",
#            "-e", "DB_HOST=postgres", "associate-associate"]
#
# Use this script as the "command" in MCP client configuration when you don't have Go installed.
# It runs the associate MCP server via Docker, communicating over stdin/stdout.

$ErrorActionPreference = "Stop"

$SCRIPT_DIR = Split-Path -Parent $MyInvocation.MyCommand.Path
$PROJECT_ROOT = Split-Path -Parent $SCRIPT_DIR

Set-Location $PROJECT_ROOT

# Check if PostgreSQL is running, start if not
$postgresRunning = docker ps --format "{{.Names}}" 2>$null | Select-String -Pattern "^associate-postgres$" -Quiet

if (-not $postgresRunning) {
    # Start PostgreSQL in the background
    docker-compose up -d postgres 2>&1 | Out-Host
    
    # Wait for PostgreSQL to be healthy (max 60 seconds)
    $maxAttempts = 30
    for ($i = 1; $i -le $maxAttempts; $i++) {
        try {
            docker exec associate-postgres pg_isready -U associate -d associate 2>&1 | Out-Null
            break
        }
        catch {
            Start-Sleep -Seconds 2
        }
    }
}

# Run associate in stdio mode, connecting to the postgres container network
# --rm: Remove container after exit
# -T: Don't allocate a pseudo-TTY (important for stdio mode)
# --entrypoint: Override the default -http entrypoint to run in stdio mode
# Logs go to stderr, MCP protocol goes to stdin/stdout
docker-compose run --rm -T --entrypoint /associate associate
