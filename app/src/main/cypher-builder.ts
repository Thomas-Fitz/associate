/**
 * Cypher query builder utilities.
 * 
 * This module provides reusable query patterns for the Apache AGE graph database.
 * It encapsulates the relationship directions and query structure to ensure
 * consistency across all database operations.
 */

import { NodeLabel, RelationType, RelationProperty } from './graph-schema'
import { escapeCypherString } from './database'

/**
 * Query patterns for Plan operations.
 */
export const PlanQueries = {
  /**
   * Match a plan by ID.
   */
  matchById(planId: string): string {
    return `MATCH (p:${NodeLabel.Plan} {id: '${escapeCypherString(planId)}'})`
  },

  /**
   * Get task count for a plan using the correct relationship direction.
   * Tasks link TO plans: (Task)-[:PART_OF]->(Plan)
   */
  withTaskCount(): string {
    return `OPTIONAL MATCH (t:${NodeLabel.Task})-[:${RelationType.PartOf}]->(p)
      WITH p, count(t) as task_count`
  },

  /**
   * Build a list plans query with optional filters.
   */
  list(options?: { status?: string; search?: string; limit?: number; offset?: number }): string {
    let query = `MATCH (p:${NodeLabel.Plan})`
    const conditions: string[] = []

    if (options?.status && options.status !== 'all') {
      conditions.push(`p.status = '${escapeCypherString(options.status)}'`)
    }

    if (options?.search) {
      const search = escapeCypherString(options.search.toLowerCase())
      conditions.push(`(toLower(p.name) CONTAINS '${search}' OR toLower(p.description) CONTAINS '${search}')`)
    }

    if (conditions.length > 0) {
      query += ` WHERE ${conditions.join(' AND ')}`
    }

    // Task count using correct direction: (Task)-[:PART_OF]->(Plan)
    query += `
      OPTIONAL MATCH (t:${NodeLabel.Task})-[:${RelationType.PartOf}]->(p)
      WITH p, count(t) as task_count
      RETURN p, task_count
      ORDER BY p.updated_at DESC
    `

    if (options?.limit) {
      query += ` LIMIT ${options.limit}`
    }
    if (options?.offset) {
      query += ` SKIP ${options.offset}`
    }

    return query
  }
}

/**
 * Query patterns for Task operations.
 */
export const TaskQueries = {
  /**
   * Match a task by ID.
   */
  matchById(taskId: string): string {
    return `MATCH (t:${NodeLabel.Task} {id: '${escapeCypherString(taskId)}'})`
  },

  /**
   * Get all tasks for a plan with their positions and relationships.
   * Uses correct direction: (Task)-[:PART_OF]->(Plan)
   */
  listByPlan(planId: string): string {
    return `
      MATCH (t:${NodeLabel.Task})-[r:${RelationType.PartOf}]->(p:${NodeLabel.Plan} {id: '${escapeCypherString(planId)}'})
      OPTIONAL MATCH (t)-[:${RelationType.DependsOn}]->(dep:${NodeLabel.Task})
      OPTIONAL MATCH (t)-[:${RelationType.Blocks}]->(blk:${NodeLabel.Task})
      WITH t, r.${RelationProperty.Position} as position, collect(DISTINCT dep.id) as depends_on, collect(DISTINCT blk.id) as blocks
      RETURN t, position, depends_on, blocks
      ORDER BY position
    `
  },

  /**
   * Get max position of tasks in a plan.
   * Uses correct direction: (Task)-[:PART_OF]->(Plan)
   */
  maxPositionInPlan(planId: string): string {
    return `
      MATCH (t:${NodeLabel.Task})-[r:${RelationType.PartOf}]->(p:${NodeLabel.Plan} {id: '${escapeCypherString(planId)}'})
      RETURN max(r.${RelationProperty.Position}) as max_pos
    `
  },

  /**
   * Link a task to a plan with position.
   * Uses correct direction: (Task)-[:PART_OF]->(Plan)
   */
  linkToPlan(taskId: string, planId: string, position: number): string {
    return `
      MATCH (t:${NodeLabel.Task} {id: '${escapeCypherString(taskId)}'}), (p:${NodeLabel.Plan} {id: '${escapeCypherString(planId)}'})
      CREATE (t)-[:${RelationType.PartOf} {${RelationProperty.Position}: ${position}}]->(p)
      RETURN t
    `
  },

  /**
   * Update a task's position within a plan.
   * Uses correct direction: (Task)-[:PART_OF]->(Plan)
   */
  updatePosition(taskId: string, planId: string, position: number): string {
    return `
      MATCH (t:${NodeLabel.Task} {id: '${escapeCypherString(taskId)}'})-[r:${RelationType.PartOf}]->(p:${NodeLabel.Plan} {id: '${escapeCypherString(planId)}'})
      SET r.${RelationProperty.Position} = ${position}
      RETURN t
    `
  }
}

/**
 * Query patterns for dependency operations.
 */
export const DependencyQueries = {
  /**
   * Check for circular dependency.
   */
  checkCircular(sourceTaskId: string, targetTaskId: string): string {
    return `
      MATCH path = (target:${NodeLabel.Task} {id: '${escapeCypherString(targetTaskId)}'})-[:${RelationType.DependsOn}*]->(source:${NodeLabel.Task} {id: '${escapeCypherString(sourceTaskId)}'})
      RETURN count(path) as cycle_count
    `
  },

  /**
   * Create a dependency between tasks.
   */
  create(sourceTaskId: string, targetTaskId: string): string {
    return `
      MATCH (source:${NodeLabel.Task} {id: '${escapeCypherString(sourceTaskId)}'}), (target:${NodeLabel.Task} {id: '${escapeCypherString(targetTaskId)}'})
      MERGE (source)-[:${RelationType.DependsOn}]->(target)
      RETURN source
    `
  },

  /**
   * Delete a dependency between tasks.
   */
  delete(sourceTaskId: string, targetTaskId: string, relationType: 'DEPENDS_ON' | 'BLOCKS' = 'DEPENDS_ON'): string {
    return `
      MATCH (source:${NodeLabel.Task} {id: '${escapeCypherString(sourceTaskId)}'})-[r:${relationType}]->(target:${NodeLabel.Task} {id: '${escapeCypherString(targetTaskId)}'})
      DELETE r
      RETURN source
    `
  }
}
