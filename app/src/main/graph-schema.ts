/**
 * Graph schema constants and utilities.
 * 
 * This module defines the graph database schema used by both the MCP server (Go)
 * and the Electron app (TypeScript). Keeping these in sync is critical for
 * cross-component compatibility.
 * 
 * Schema Convention (matching Go backend):
 * - Tasks link TO plans: (Task)-[:PART_OF {position}]->(Plan)
 * - Task dependencies: (Task)-[:DEPENDS_ON]->(Task)
 * - Task blocking: (Task)-[:BLOCKS]->(Task)
 */

// Node labels
export const NodeLabel = {
  Zone: 'Zone',
  Plan: 'Plan',
  Task: 'Task',
  Memory: 'Memory'
} as const

export type NodeLabelType = (typeof NodeLabel)[keyof typeof NodeLabel]

// Relationship types
export const RelationType = {
  /** Plan/Memory belongs to a Zone: (Plan)-[:BELONGS_TO]->(Zone), (Memory)-[:BELONGS_TO]->(Zone) */
  BelongsTo: 'BELONGS_TO',
  /** Task belongs to a Plan: (Task)-[:PART_OF]->(Plan) */
  PartOf: 'PART_OF',
  /** Task depends on another Task: (Task)-[:DEPENDS_ON]->(Task) */
  DependsOn: 'DEPENDS_ON',
  /** Task blocks another Task: (Task)-[:BLOCKS]->(Task) */
  Blocks: 'BLOCKS',
  /** General relation between nodes */
  RelatesTo: 'RELATES_TO',
  /** Node references another node */
  References: 'REFERENCES',
  /** Node follows another in sequence */
  Follows: 'FOLLOWS',
  /** Node implements another */
  Implements: 'IMPLEMENTS'
} as const

export type RelationTypeValue = (typeof RelationType)[keyof typeof RelationType]

// Relationship property names
export const RelationProperty = {
  Position: 'position'
} as const

// Node property names
export const NodeProperty = {
  Id: 'id',
  NodeType: 'node_type',
  Content: 'content',
  Name: 'name',
  Description: 'description',
  Status: 'status',
  Metadata: 'metadata',
  Tags: 'tags',
  CreatedAt: 'created_at',
  UpdatedAt: 'updated_at'
} as const
