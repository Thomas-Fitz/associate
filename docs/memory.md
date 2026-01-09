## Direct Neo4j Queries

You can query the Neo4j database directly using Cypher for advanced analysis. Access the Neo4j Browser at http://localhost:7474 when running via Docker.

### Find Newly Created Memories with Linked Memories

Find memories created in the last 7 days along with all their related memories:

```cypher
MATCH (m:Memory)
WHERE m.created_at >= datetime() - duration('P7D')
OPTIONAL MATCH (m)-[r]-(related:Memory)
RETURN m.id AS id,
       m.type AS type,
       m.content AS content,
       m.created_at AS created_at,
       collect(DISTINCT {
         id: related.id,
         type: related.type,
         relationship: type(r)
       }) AS linked_memories
ORDER BY m.created_at DESC
```

### Find Memories Not Updated in the Last 30 Days

Identify stale memories that may need review or cleanup:

```cypher
MATCH (m:Memory)
WHERE m.updated_at < datetime() - duration('P30D')
RETURN m.id AS id,
       m.type AS type,
       m.content AS content,
       m.updated_at AS last_updated,
       duration.between(m.updated_at, datetime()).days AS days_since_update
ORDER BY m.updated_at ASC
```

### Show All Memories

List all memories in the database:

```cypher
MATCH (m:Memory)
RETURN m.id AS id,
       m.type AS type,
       m.content AS content,
       m.tags AS tags,
       m.created_at AS created_at,
       m.updated_at AS updated_at
ORDER BY m.created_at DESC
```

To include relationship counts:

```cypher
MATCH (m:Memory)
OPTIONAL MATCH (m)-[r]-()
RETURN m.id AS id,
       m.type AS type,
       m.content AS content,
       count(r) AS relationship_count
ORDER BY relationship_count DESC
```
