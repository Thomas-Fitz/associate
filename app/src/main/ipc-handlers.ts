import { ipcMain } from 'electron'
import {
  executeCypher,
  executeCypherInTransaction,
  getClient,
  escapeCypherString,
  metadataToJSON,
  tagsToCypherList,
  parseAGTypeProperties
} from './database'
import { NodeLabel, RelationType, RelationProperty } from './graph-schema'
import { PlanQueries, TaskQueries, DependencyQueries } from './cypher-builder'
import type {
  Plan,
  PlanStatus,
  Task,
  TaskInPlan,
  TaskStatus,
  ListPlansOptions,
  CreateTaskOptions,
  UpdateTaskOptions,
  PlanWithTasks
} from '../renderer/types'

// ============================================================================
// Result Parsers - Convert AGE query results to typed objects
// ============================================================================

/**
 * Parse metadata from AGE result (can be string, object, or undefined).
 */
function parseMetadata(metadata: unknown): Record<string, unknown> {
  if (!metadata) return {}
  if (typeof metadata === 'object') return metadata as Record<string, unknown>
  if (typeof metadata === 'string') {
    if (metadata === '' || metadata === '{}') return {}
    try {
      return JSON.parse(metadata) as Record<string, unknown>
    } catch {
      return {}
    }
  }
  return {}
}

/**
 * Parse AGE array result (can be string or array).
 */
function parseAGArray(value: unknown): string[] {
  if (!value) return []
  if (Array.isArray(value)) {
    return value.map(v => String(v)).filter(v => v && v !== 'null')
  }
  if (typeof value === 'string') {
    try {
      const parsed = JSON.parse(value)
      if (Array.isArray(parsed)) {
        return parsed.map(v => String(v)).filter(v => v && v !== 'null')
      }
    } catch {
      // Not valid JSON
    }
  }
  return []
}

/**
 * Convert AGE result row to Plan object.
 */
function rowToPlan(row: Record<string, unknown>): Plan {
  const props = parseAGTypeProperties(row.p || row.plan || row.result)
  return {
    id: String(props.id || ''),
    name: String(props.name || ''),
    description: String(props.description || ''),
    status: (props.status as PlanStatus) || 'draft',
    metadata: parseMetadata(props.metadata),
    tags: Array.isArray(props.tags) ? props.tags : [],
    createdAt: String(props.created_at || props.createdAt || ''),
    updatedAt: String(props.updated_at || props.updatedAt || ''),
    taskCount: typeof props.task_count === 'number' ? props.task_count : undefined
  }
}

/**
 * Convert AGE result row to Task object.
 */
function rowToTask(row: Record<string, unknown>): Task {
  const props = parseAGTypeProperties(row.t || row.task || row.result)
  return {
    id: String(props.id || ''),
    content: String(props.content || ''),
    status: (props.status as TaskStatus) || 'pending',
    metadata: parseMetadata(props.metadata),
    tags: Array.isArray(props.tags) ? props.tags : [],
    createdAt: String(props.created_at || props.createdAt || ''),
    updatedAt: String(props.updated_at || props.updatedAt || '')
  }
}

/**
 * Convert AGE result row to TaskInPlan object (includes position and relationships).
 */
function rowToTaskInPlan(row: Record<string, unknown>): TaskInPlan {
  const task = rowToTask(row)

  // Parse position - AGE might return as string
  let position = 0
  if (typeof row.position === 'number') {
    position = row.position
  } else if (typeof row.position === 'string') {
    position = parseFloat(row.position) || 0
  }

  // Parse dependency arrays
  const dependsOn = parseAGArray(row.depends_on)
  const blocks = parseAGArray(row.blocks)

  console.log('rowToTaskInPlan:', {
    taskId: task.id,
    position,
    dependsOn,
    blocks,
    rawDepsOn: row.depends_on,
    rawBlocks: row.blocks
  })

  return {
    ...task,
    position,
    dependsOn,
    blocks
  }
}

/**
 * Parse task count from AGE result (can be number or string).
 */
function parseTaskCount(value: unknown): number {
  if (typeof value === 'number') return value
  if (typeof value === 'string') return parseInt(value, 10) || 0
  return 0
}

// ============================================================================
// IPC Handlers Setup
// ============================================================================

export function setupIpcHandlers(): void {
  // --------------------------------------------------------------------------
  // Plan Handlers
  // --------------------------------------------------------------------------

  /**
   * List plans with optional filtering by status and search term.
   */
  ipcMain.handle('db:plans:list', async (_event, options?: ListPlansOptions) => {
    console.log('db:plans:list called with options:', options)

    const query = PlanQueries.list({
      status: options?.status,
      search: options?.search,
      limit: options?.limit,
      offset: options?.offset
    })

    console.log('Executing query:', query)
    const rows = await executeCypher<Record<string, unknown>>(query, 'p agtype, task_count agtype')
    console.log('Query returned rows:', rows.length, rows)

    const plans = rows.map(row => {
      console.log('Processing row:', row)
      const plan = rowToPlan(row)
      plan.taskCount = parseTaskCount(row.task_count)
      console.log('Converted to plan:', plan)
      return plan
    })

    console.log('Returning plans:', plans)
    return plans
  })

  /**
   * Get a single plan with all its tasks.
   */
  ipcMain.handle('db:plans:get', async (_event, planId: string): Promise<PlanWithTasks | null> => {
    const escapedId = escapeCypherString(planId)

    // Get the plan
    const planQuery = `MATCH (p:${NodeLabel.Plan} {id: '${escapedId}'}) RETURN p`
    const planRows = await executeCypher<Record<string, unknown>>(planQuery, 'p agtype')

    if (planRows.length === 0) {
      return null
    }

    const plan = rowToPlan(planRows[0])

    // Get tasks with positions and relationships using correct direction
    const tasksQuery = TaskQueries.listByPlan(planId)
    const taskRows = await executeCypher<Record<string, unknown>>(
      tasksQuery,
      't agtype, position agtype, depends_on agtype, blocks agtype'
    )

    const tasks: TaskInPlan[] = taskRows.map(row => rowToTaskInPlan(row))

    return {
      ...plan,
      tasks
    }
  })

  // --------------------------------------------------------------------------
  // Task Handlers
  // --------------------------------------------------------------------------

  /**
   * Create a new task and link it to a plan.
   * Uses correct relationship direction: (Task)-[:PART_OF]->(Plan)
   */
  ipcMain.handle('db:tasks:create', async (_event, options: CreateTaskOptions): Promise<TaskInPlan> => {
    const client = await getClient()

    try {
      await client.query('BEGIN')

      const taskId = crypto.randomUUID()
      const now = new Date().toISOString()
      const escapedContent = escapeCypherString(options.content)
      const status = options.status || 'pending'
      const metadata = metadataToJSON(options.metadata || {})
      const tags = tagsToCypherList(options.tags || [])

      // Calculate position if not provided
      let position = options.position
      if (position === undefined) {
        const maxPosQuery = TaskQueries.maxPositionInPlan(options.planId)
        const posRows = await executeCypherInTransaction<{ max_pos: unknown }>(
          client,
          maxPosQuery,
          'max_pos agtype'
        )
        const maxPos = posRows[0]?.max_pos
        position = (typeof maxPos === 'number' ? maxPos : 0) + 1000
      }

      // Create the task node
      const createQuery = `
        CREATE (t:${NodeLabel.Task} {
          id: '${taskId}',
          node_type: 'Task',
          content: '${escapedContent}',
          status: '${status}',
          metadata: '${metadata}',
          tags: ${tags},
          created_at: '${now}',
          updated_at: '${now}'
        })
        RETURN t
      `
      await executeCypherInTransaction(client, createQuery, 't agtype')

      // Link task to plan using correct direction: (Task)-[:PART_OF]->(Plan)
      const linkQuery = TaskQueries.linkToPlan(taskId, options.planId, position)
      const linkRows = await executeCypherInTransaction<Record<string, unknown>>(
        client,
        linkQuery,
        't agtype'
      )

      await client.query('COMMIT')

      const task = rowToTask(linkRows[0])
      return {
        ...task,
        position,
        dependsOn: [],
        blocks: []
      }
    } catch (err) {
      await client.query('ROLLBACK')
      throw err
    } finally {
      client.release()
    }
  })

  /**
   * Update a task's properties.
   */
  ipcMain.handle(
    'db:tasks:update',
    async (_event, taskId: string, options: UpdateTaskOptions): Promise<Task> => {
      const escapedId = escapeCypherString(taskId)
      const now = new Date().toISOString()

      const sets: string[] = [`t.updated_at = '${now}'`]

      if (options.content !== undefined) {
        sets.push(`t.content = '${escapeCypherString(options.content)}'`)
      }
      if (options.status !== undefined) {
        sets.push(`t.status = '${options.status}'`)
      }
      if (options.metadata !== undefined) {
        sets.push(`t.metadata = '${metadataToJSON(options.metadata)}'`)
      }
      if (options.tags !== undefined) {
        sets.push(`t.tags = ${tagsToCypherList(options.tags)}`)
      }

      const query = `
      MATCH (t:${NodeLabel.Task} {id: '${escapedId}'})
      SET ${sets.join(', ')}
      RETURN t
    `

      const rows = await executeCypher<Record<string, unknown>>(query, 't agtype')

      if (rows.length === 0) {
        throw new Error(`Task not found: ${taskId}`)
      }

      return rowToTask(rows[0])
    }
  )

  /**
   * Delete a task and all its relationships.
   */
  ipcMain.handle('db:tasks:delete', async (_event, taskId: string): Promise<void> => {
    const escapedId = escapeCypherString(taskId)

    const query = `
      MATCH (t:${NodeLabel.Task} {id: '${escapedId}'})
      DETACH DELETE t
    `

    await executeCypher(query, 'result agtype')
  })

  /**
   * Reorder tasks within a plan by updating their positions.
   * Uses correct relationship direction: (Task)-[:PART_OF]->(Plan)
   */
  ipcMain.handle(
    'db:tasks:reorder',
    async (_event, planId: string, taskIds: string[]): Promise<void> => {
      const client = await getClient()

      try {
        await client.query('BEGIN')

        // Update each task's position using the correct relationship direction
        for (let i = 0; i < taskIds.length; i++) {
          const position = (i + 1) * 1000 // Use 1000, 2000, 3000... for easy insertion
          const query = TaskQueries.updatePosition(taskIds[i], planId, position)
          await executeCypherInTransaction(client, query, 't agtype')
        }

        await client.query('COMMIT')
      } catch (err) {
        await client.query('ROLLBACK')
        throw err
      } finally {
        client.release()
      }
    }
  )

  // --------------------------------------------------------------------------
  // Dependency Handlers
  // --------------------------------------------------------------------------

  /**
   * Create a dependency between two tasks.
   */
  ipcMain.handle(
    'db:dependencies:create',
    async (_event, sourceTaskId: string, targetTaskId: string): Promise<void> => {
      // Check for circular dependency
      const circularCheck = DependencyQueries.checkCircular(sourceTaskId, targetTaskId)
      const checkRows = await executeCypher<{ cycle_count: unknown }>(
        circularCheck,
        'cycle_count agtype'
      )
      const cycleCount = checkRows[0]?.cycle_count
      if (typeof cycleCount === 'number' && cycleCount > 0) {
        throw new Error('Cannot create circular dependency')
      }

      // Create the dependency
      const query = DependencyQueries.create(sourceTaskId, targetTaskId)
      await executeCypher(query, 'source agtype')
    }
  )

  /**
   * Delete a dependency between two tasks.
   */
  ipcMain.handle(
    'db:dependencies:delete',
    async (
      _event,
      sourceTaskId: string,
      targetTaskId: string,
      relationshipType: 'DEPENDS_ON' | 'BLOCKS' = 'DEPENDS_ON'
    ): Promise<void> => {
      const query = DependencyQueries.delete(sourceTaskId, targetTaskId, relationshipType)
      await executeCypher(query, 'source agtype')
    }
  )
}
