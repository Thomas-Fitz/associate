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
 * Query patterns for Zone operations.
 */
export const ZoneQueries = {
  /**
   * List zones with optional search filter.
   * Returns zones sorted by updated_at DESC with plan, task, and memory counts.
   */
  list(options?: { search?: string; limit?: number }): string {
    let query = `MATCH (z:${NodeLabel.Zone})`
    
    if (options?.search) {
      const search = escapeCypherString(options.search.toLowerCase())
      query += ` WHERE toLower(z.name) CONTAINS '${search}' OR toLower(z.description) CONTAINS '${search}'`
    }
    
    query += `
      OPTIONAL MATCH (p:${NodeLabel.Plan})-[:${RelationType.BelongsTo}]->(z)
      OPTIONAL MATCH (t:${NodeLabel.Task})-[:${RelationType.PartOf}]->(p)
      OPTIONAL MATCH (m:${NodeLabel.Memory})-[:${RelationType.BelongsTo}]->(z)
      WITH z, count(DISTINCT p) as plan_count, count(DISTINCT t) as task_count, count(DISTINCT m) as memory_count
      RETURN z, plan_count, task_count, memory_count
      ORDER BY z.updated_at DESC
    `
    
    if (options?.limit) {
      query += ` LIMIT ${options.limit}`
    }
    
    return query
  },

  /**
   * Get a single zone by ID.
   */
  getById(zoneId: string): string {
    return `
      MATCH (z:${NodeLabel.Zone} {id: '${escapeCypherString(zoneId)}'})
      OPTIONAL MATCH (p:${NodeLabel.Plan})-[:${RelationType.BelongsTo}]->(z)
      OPTIONAL MATCH (t:${NodeLabel.Task})-[:${RelationType.PartOf}]->(p)
      OPTIONAL MATCH (m:${NodeLabel.Memory})-[:${RelationType.BelongsTo}]->(z)
      WITH z, count(DISTINCT p) as plan_count, count(DISTINCT t) as task_count, count(DISTINCT m) as memory_count
      RETURN z, plan_count, task_count, memory_count
    `
  },

  /**
   * Get zone with full contents (plans with tasks, and memories).
   */
  getWithContents(zoneId: string): string {
    return `
      MATCH (z:${NodeLabel.Zone} {id: '${escapeCypherString(zoneId)}'})
      RETURN z
    `
  },

  /**
   * Get all plans in a zone with their tasks.
   * Note: We filter out null tasks from the collect to avoid empty wrapper objects.
   * Tasks are ordered by their position on the PART_OF relationship.
   */
  getPlansWithTasks(zoneId: string): string {
    return `
      MATCH (p:${NodeLabel.Plan})-[:${RelationType.BelongsTo}]->(z:${NodeLabel.Zone} {id: '${escapeCypherString(zoneId)}'})
      OPTIONAL MATCH (t:${NodeLabel.Task})-[r:${RelationType.PartOf}]->(p)
      OPTIONAL MATCH (t)-[:${RelationType.DependsOn}]->(dep:${NodeLabel.Task})
      OPTIONAL MATCH (t)-[:${RelationType.Blocks}]->(blk:${NodeLabel.Task})
      WITH p, t, r, collect(DISTINCT dep.id) as depends_on, collect(DISTINCT blk.id) as blocks
      ORDER BY r.${RelationProperty.Position}
      WITH p, CASE WHEN t IS NOT NULL THEN {task: t, depends_on: depends_on, blocks: blocks} ELSE NULL END as task_obj
      WITH p, collect(task_obj) as all_tasks
      RETURN p, [x IN all_tasks WHERE x IS NOT NULL] as tasks
      ORDER BY p.updated_at DESC
    `
  },

  /**
   * Get all memories in a zone.
   */
  getMemories(zoneId: string): string {
    return `
      MATCH (m:${NodeLabel.Memory})-[:${RelationType.BelongsTo}]->(z:${NodeLabel.Zone} {id: '${escapeCypherString(zoneId)}'})
      RETURN m
      ORDER BY m.updated_at DESC
    `
  },

  /**
   * Create a new zone.
   */
  create(options: { id: string; name: string; description?: string; metadata?: string; tags?: string }): string {
    const now = new Date().toISOString()
    return `
      CREATE (z:${NodeLabel.Zone} {
        id: '${escapeCypherString(options.id)}',
        node_type: 'Zone',
        name: '${escapeCypherString(options.name)}',
        description: '${escapeCypherString(options.description || '')}',
        metadata: '${options.metadata || '{}'}',
        tags: ${options.tags || '[]'},
        created_at: '${now}',
        updated_at: '${now}'
      })
      RETURN z
    `
  },

  /**
   * Update a zone.
   */
  update(zoneId: string, sets: string[]): string {
    return `
      MATCH (z:${NodeLabel.Zone} {id: '${escapeCypherString(zoneId)}'})
      SET ${sets.join(', ')}
      RETURN z
    `
  },

  /**
   * Delete a zone and all its contents (cascade delete).
   */
  delete(zoneId: string): string {
    return `
      MATCH (z:${NodeLabel.Zone} {id: '${escapeCypherString(zoneId)}'})
      OPTIONAL MATCH (p:${NodeLabel.Plan})-[:${RelationType.BelongsTo}]->(z)
      OPTIONAL MATCH (t:${NodeLabel.Task})-[:${RelationType.PartOf}]->(p)
      OPTIONAL MATCH (m:${NodeLabel.Memory})-[:${RelationType.BelongsTo}]->(z)
      DETACH DELETE z, p, t, m
    `
  }
}

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
  list(options?: { status?: string; search?: string; limit?: number; offset?: number; zoneId?: string }): string {
    let query = `MATCH (p:${NodeLabel.Plan})`
    const conditions: string[] = []

    if (options?.zoneId) {
      query = `MATCH (p:${NodeLabel.Plan})-[:${RelationType.BelongsTo}]->(z:${NodeLabel.Zone} {id: '${escapeCypherString(options.zoneId)}'})`
    }

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
  },

  /**
   * Create a new plan.
   */
  create(options: { id: string; name: string; description?: string; status?: string; metadata?: string; tags?: string }): string {
    const now = new Date().toISOString()
    return `
      CREATE (p:${NodeLabel.Plan} {
        id: '${escapeCypherString(options.id)}',
        node_type: 'Plan',
        name: '${escapeCypherString(options.name)}',
        description: '${escapeCypherString(options.description || '')}',
        status: '${escapeCypherString(options.status || 'draft')}',
        metadata: '${options.metadata || '{}'}',
        tags: ${options.tags || '[]'},
        created_at: '${now}',
        updated_at: '${now}'
      })
      RETURN p
    `
  },

  /**
   * Link a plan to a zone.
   */
  linkToZone(planId: string, zoneId: string): string {
    return `
      MATCH (p:${NodeLabel.Plan} {id: '${escapeCypherString(planId)}'}), (z:${NodeLabel.Zone} {id: '${escapeCypherString(zoneId)}'})
      CREATE (p)-[:${RelationType.BelongsTo}]->(z)
      RETURN p
    `
  },

  /**
   * Move a plan to a different zone.
   */
  moveToZone(planId: string, newZoneId: string): string {
    return `
      MATCH (p:${NodeLabel.Plan} {id: '${escapeCypherString(planId)}'})-[r:${RelationType.BelongsTo}]->(oldZ:${NodeLabel.Zone})
      DELETE r
      WITH p
      MATCH (newZ:${NodeLabel.Zone} {id: '${escapeCypherString(newZoneId)}'})
      CREATE (p)-[:${RelationType.BelongsTo}]->(newZ)
      RETURN p
    `
  },

  /**
   * Update a plan.
   */
  update(planId: string, sets: string[]): string {
    return `
      MATCH (p:${NodeLabel.Plan} {id: '${escapeCypherString(planId)}'})
      SET ${sets.join(', ')}
      RETURN p
    `
  },

  /**
   * Delete a plan and all its tasks.
   */
  delete(planId: string): string {
    return `
      MATCH (p:${NodeLabel.Plan} {id: '${escapeCypherString(planId)}'})
      OPTIONAL MATCH (t:${NodeLabel.Task})-[:${RelationType.PartOf}]->(p)
      DETACH DELETE p, t
    `
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
  },

  /**
   * Move a task to a different plan.
   * Removes the old PART_OF relationship and creates a new one.
   */
  moveToPlan(taskId: string, newPlanId: string, position: number): string {
    return `
      MATCH (t:${NodeLabel.Task} {id: '${escapeCypherString(taskId)}'})-[r:${RelationType.PartOf}]->(oldP:${NodeLabel.Plan})
      DELETE r
      WITH t
      MATCH (newP:${NodeLabel.Plan} {id: '${escapeCypherString(newPlanId)}'})
      CREATE (t)-[:${RelationType.PartOf} {${RelationProperty.Position}: ${position}}]->(newP)
      RETURN t
    `
  },

  /**
   * Update a task's properties.
   */
  update(taskId: string, sets: string[]): string {
    return `
      MATCH (t:${NodeLabel.Task} {id: '${escapeCypherString(taskId)}'})
      SET ${sets.join(', ')}
      RETURN t
    `
  },

  /**
   * Delete a task and all its relationships.
   */
  delete(taskId: string): string {
    return `
      MATCH (t:${NodeLabel.Task} {id: '${escapeCypherString(taskId)}'})
      DETACH DELETE t
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

/**
 * Query patterns for Memory operations.
 */
export const MemoryQueries = {
  /**
   * Create a new memory.
   */
  create(options: { id: string; type: string; content: string; metadata?: string; tags?: string }): string {
    const now = new Date().toISOString()
    return `
      CREATE (m:${NodeLabel.Memory} {
        id: '${escapeCypherString(options.id)}',
        node_type: '${escapeCypherString(options.type)}',
        content: '${escapeCypherString(options.content)}',
        metadata: '${options.metadata || '{}'}',
        tags: ${options.tags || '[]'},
        created_at: '${now}',
        updated_at: '${now}'
      })
      RETURN m
    `
  },

  /**
   * Link a memory to a zone.
   */
  linkToZone(memoryId: string, zoneId: string): string {
    return `
      MATCH (m:${NodeLabel.Memory} {id: '${escapeCypherString(memoryId)}'}), (z:${NodeLabel.Zone} {id: '${escapeCypherString(zoneId)}'})
      CREATE (m)-[:${RelationType.BelongsTo}]->(z)
      RETURN m
    `
  },

  /**
   * Update a memory.
   */
  update(memoryId: string, sets: string[]): string {
    return `
      MATCH (m:${NodeLabel.Memory} {id: '${escapeCypherString(memoryId)}'})
      SET ${sets.join(', ')}
      RETURN m
    `
  },

  /**
   * Delete a memory.
   */
  delete(memoryId: string): string {
    return `
      MATCH (m:${NodeLabel.Memory} {id: '${escapeCypherString(memoryId)}'})
      DETACH DELETE m
    `
  },

  /**
   * Create a RELATES_TO relationship from memory to another node.
   */
  linkTo(memoryId: string, targetId: string): string {
    return `
      MATCH (m:${NodeLabel.Memory} {id: '${escapeCypherString(memoryId)}'})
      MATCH (target {id: '${escapeCypherString(targetId)}'})
      WHERE label(target) IN ['${NodeLabel.Zone}', '${NodeLabel.Plan}', '${NodeLabel.Task}']
      MERGE (m)-[:${RelationType.RelatesTo}]->(target)
      RETURN m
    `
  },

  /**
   * Remove a RELATES_TO relationship.
   */
  unlinkFrom(memoryId: string, targetId: string): string {
    return `
      MATCH (m:${NodeLabel.Memory} {id: '${escapeCypherString(memoryId)}'})-[r:${RelationType.RelatesTo}]->(target {id: '${escapeCypherString(targetId)}'})
      DELETE r
      RETURN m
    `
  }
}
