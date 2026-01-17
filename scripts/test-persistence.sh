#!/bin/bash
set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
NC='\033[0m' # No Color

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
cd "$PROJECT_ROOT"

echo "=== Neo4j Persistence Test ==="
echo "This test verifies that data persists across docker-compose down"
echo ""

# Step 1: Start services
echo "Step 1: Starting services..."
docker-compose up -d --build
sleep 5

# Wait for Neo4j to be healthy
echo "Waiting for Neo4j to be healthy..."
for i in {1..30}; do
    if curl -sf http://localhost:7474 > /dev/null 2>&1; then
        echo "Neo4j is ready!"
        break
    fi
    sleep 2
done

# Step 2: Create a test memory
TEST_ID="persist-test-$(date +%s)"
echo ""
echo "Step 2: Creating test memory with ID: $TEST_ID"
curl -s -X POST http://localhost:7474/db/neo4j/tx/commit \
    -H "Content-Type: application/json" \
    -H "Authorization: Basic $(echo -n 'neo4j:password' | base64)" \
    -d "{
        \"statements\": [{
            \"statement\": \"CREATE (m:Memory {id: '\$id', content: 'Persistence test', created_at: datetime()}) RETURN m.id\",
            \"parameters\": {\"id\": \"$TEST_ID\"}
        }]
    }" > /dev/null

# Step 3: Verify it exists
echo ""
echo "Step 3: Verifying memory exists..."
RESULT=$(curl -s -X POST http://localhost:7474/db/neo4j/tx/commit \
    -H "Content-Type: application/json" \
    -H "Authorization: Basic $(echo -n 'neo4j:password' | base64)" \
    -d "{
        \"statements\": [{
            \"statement\": \"MATCH (m:Memory {id: '\$id'}) RETURN m.id\",
            \"parameters\": {\"id\": \"$TEST_ID\"}
        }]
    }")

if echo "$RESULT" | grep -q "$TEST_ID"; then
    echo -e "${GREEN}✓ Memory found before shutdown${NC}"
else
    echo -e "${RED}✗ Memory NOT found before shutdown${NC}"
    exit 1
fi

# Step 4: Stop without -v flag (preserves volumes)
echo ""
echo "Step 4: Running docker-compose down (without -v to preserve data)..."
docker-compose down

# Step 5: Start again
echo ""
echo "Step 5: Starting services again..."
docker-compose up -d
sleep 5

# Wait for Neo4j
echo "Waiting for Neo4j to be healthy..."
for i in {1..30}; do
    if curl -sf http://localhost:7474 > /dev/null 2>&1; then
        echo "Neo4j is ready!"
        break
    fi
    sleep 2
done

# Step 6: Verify data persisted
echo ""
echo "Step 6: Verifying memory persisted after restart..."
RESULT=$(curl -s -X POST http://localhost:7474/db/neo4j/tx/commit \
    -H "Content-Type: application/json" \
    -H "Authorization: Basic $(echo -n 'neo4j:password' | base64)" \
    -d "{
        \"statements\": [{
            \"statement\": \"MATCH (m:Memory {id: '\$id'}) RETURN m.id\",
            \"parameters\": {\"id\": \"$TEST_ID\"}
        }]
    }")

if echo "$RESULT" | grep -q "$TEST_ID"; then
    echo -e "${GREEN}✓ SUCCESS: Memory persisted after docker-compose down!${NC}"
    echo ""
    echo "Docker volumes work correctly. Data persists across container restarts."
else
    echo -e "${RED}✗ FAILURE: Memory was lost after restart${NC}"
    exit 1
fi

# Cleanup
echo ""
echo "Cleaning up test memory..."
curl -s -X POST http://localhost:7474/db/neo4j/tx/commit \
    -H "Content-Type: application/json" \
    -H "Authorization: Basic $(echo -n 'neo4j:password' | base64)" \
    -d "{
        \"statements\": [{
            \"statement\": \"MATCH (m:Memory {id: '\$id'}) DELETE m\",
            \"parameters\": {\"id\": \"$TEST_ID\"}
        }]
    }" > /dev/null

echo ""
echo "=== Test Complete ==="
