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
import { PlanQueries, TaskQueries, DependencyQueries, ZoneQueries, MemoryQueries } from './cypher-builder'
import type {
  Plan,
  PlanStatus,
  Task,
  TaskInPlan,
  TaskStatus,
  ListPlansOptions,
  CreateTaskOptions,
  UpdateTaskOptions,
  PlanWithTasks,
  Zone,
  ZoneWithContents,
  PlanInZone,
  TaskInZone,
  MemoryInZone
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

/**
 * Convert AGE result row to Zone object.
 */
function rowToZone(row: Record<string, unknown>): Zone {
  const props = parseAGTypeProperties(row.z || row.zone || row.result)
  return {
    id: String(props.id || ''),
    name: String(props.name || ''),
    description: String(props.description || ''),
    metadata: parseMetadata(props.metadata),
    tags: Array.isArray(props.tags) ? props.tags : [],
    createdAt: String(props.created_at || props.createdAt || ''),
    updatedAt: String(props.updated_at || props.updatedAt || ''),
    planCount: parseTaskCount(row.plan_count),
    taskCount: parseTaskCount(row.task_count),
    memoryCount: parseTaskCount(row.memory_count)
  }
}

/**
 * Convert AGE result row to Memory object.
 */
function rowToMemory(row: Record<string, unknown>): MemoryInZone {
  const props = parseAGTypeProperties(row.m || row.memory || row.result)
  const metadata = parseMetadata(props.metadata)
  return {
    id: String(props.id || ''),
    type: (props.node_type as 'Note' | 'Repository' | 'Memory') || 'Memory',
    content: String(props.content || ''),
    metadata,
    tags: Array.isArray(props.tags) ? props.tags : [],
    createdAt: String(props.created_at || props.createdAt || ''),
    updatedAt: String(props.updated_at || props.updatedAt || ''),
    ui_x: typeof metadata.ui_x === 'number' ? metadata.ui_x : undefined,
    ui_y: typeof metadata.ui_y === 'number' ? metadata.ui_y : undefined,
    ui_width: typeof metadata.ui_width === 'number' ? metadata.ui_width : undefined,
    ui_height: typeof metadata.ui_height === 'number' ? metadata.ui_height : undefined
  }
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

  // --------------------------------------------------------------------------
  // Zone Handlers
  // --------------------------------------------------------------------------

  /**
   * List zones with optional search filter.
   */
  ipcMain.handle('db:zones:list', async (_event, options?: { search?: string; limit?: number }): Promise<Zone[]> => {
    console.log('db:zones:list called with options:', options)

    const query = ZoneQueries.list(options)
    console.log('Executing query:', query)
    
    const rows = await executeCypher<Record<string, unknown>>(
      query,
      'z agtype, plan_count agtype, task_count agtype, memory_count agtype'
    )
    console.log('Query returned rows:', rows.length)

    return rows.map(row => rowToZone(row))
  })

  /**
   * Get a zone by ID with counts.
   */
  ipcMain.handle('db:zones:getById', async (_event, zoneId: string): Promise<Zone | null> => {
    const query = ZoneQueries.getById(zoneId)
    const rows = await executeCypher<Record<string, unknown>>(
      query,
      'z agtype, plan_count agtype, task_count agtype, memory_count agtype'
    )

    if (rows.length === 0) {
      return null
    }

    return rowToZone(rows[0])
  })

  /**
   * Get a zone with all its contents (plans with tasks, memories).
   */
  ipcMain.handle('db:zones:get', async (_event, zoneId: string): Promise<ZoneWithContents | null> => {
    // Get the zone
    const zoneQuery = ZoneQueries.getWithContents(zoneId)
    const zoneRows = await executeCypher<Record<string, unknown>>(zoneQuery, 'z agtype')

    if (zoneRows.length === 0) {
      return null
    }

    const zoneProps = parseAGTypeProperties(zoneRows[0].z || zoneRows[0].zone || zoneRows[0].result)
    const zone: Zone = {
      id: String(zoneProps.id || ''),
      name: String(zoneProps.name || ''),
      description: String(zoneProps.description || ''),
      metadata: parseMetadata(zoneProps.metadata),
      tags: Array.isArray(zoneProps.tags) ? zoneProps.tags : [],
      createdAt: String(zoneProps.created_at || zoneProps.createdAt || ''),
      updatedAt: String(zoneProps.updated_at || zoneProps.updatedAt || '')
    }

    // Get plans with their tasks
    const plansQuery = ZoneQueries.getPlansWithTasks(zoneId)
    const planRows = await executeCypher<Record<string, unknown>>(
      plansQuery,
      'p agtype, tasks agtype'
    )

    const plans: PlanInZone[] = planRows.map(row => {
      const planProps = parseAGTypeProperties(row.p)
      const planMetadata = parseMetadata(planProps.metadata)
      
      // Parse tasks array
      let tasksData: unknown[] = []
      if (typeof row.tasks === 'string') {
        try {
          tasksData = JSON.parse(row.tasks) || []
        } catch {
          tasksData = []
        }
      } else if (Array.isArray(row.tasks)) {
        tasksData = row.tasks
      }

      const tasks: TaskInZone[] = tasksData
        .filter((t: unknown) => t && typeof t === 'object' && (t as Record<string, unknown>).task)
        .map((t: unknown) => {
          const taskObj = t as Record<string, unknown>
          const taskProps = parseAGTypeProperties(taskObj.task)
          const taskMetadata = parseMetadata(taskProps.metadata)
          
          return {
            id: String(taskProps.id || ''),
            content: String(taskProps.content || ''),
            status: (taskProps.status as TaskInZone['status']) || 'pending',
            metadata: {
              ...taskMetadata,
              ui_x: typeof taskMetadata.ui_x === 'number' ? taskMetadata.ui_x : undefined,
              ui_y: typeof taskMetadata.ui_y === 'number' ? taskMetadata.ui_y : undefined,
              ui_width: typeof taskMetadata.ui_width === 'number' ? taskMetadata.ui_width : undefined,
              ui_height: typeof taskMetadata.ui_height === 'number' ? taskMetadata.ui_height : undefined
            },
            tags: Array.isArray(taskProps.tags) ? taskProps.tags : [],
            createdAt: String(taskProps.created_at || taskProps.createdAt || ''),
            updatedAt: String(taskProps.updated_at || taskProps.updatedAt || ''),
            planId: String(planProps.id || ''),
            dependsOn: parseAGArray(taskObj.depends_on),
            blocks: parseAGArray(taskObj.blocks)
          }
        })

      return {
        id: String(planProps.id || ''),
        name: String(planProps.name || ''),
        description: String(planProps.description || ''),
        status: (planProps.status as PlanInZone['status']) || 'draft',
        metadata: {
          ...planMetadata,
          ui_x: typeof planMetadata.ui_x === 'number' ? planMetadata.ui_x : undefined,
          ui_y: typeof planMetadata.ui_y === 'number' ? planMetadata.ui_y : undefined,
          ui_width: typeof planMetadata.ui_width === 'number' ? planMetadata.ui_width : undefined,
          ui_height: typeof planMetadata.ui_height === 'number' ? planMetadata.ui_height : undefined
        },
        tags: Array.isArray(planProps.tags) ? planProps.tags : [],
        createdAt: String(planProps.created_at || planProps.createdAt || ''),
        updatedAt: String(planProps.updated_at || planProps.updatedAt || ''),
        tasks
      }
    })

    // Get memories
    const memoriesQuery = ZoneQueries.getMemories(zoneId)
    const memoryRows = await executeCypher<Record<string, unknown>>(memoriesQuery, 'm agtype')
    const memories: MemoryInZone[] = memoryRows.map(row => rowToMemory(row))

    return {
      ...zone,
      plans,
      memories,
      planCount: plans.length,
      taskCount: plans.reduce((sum, p) => sum + p.tasks.length, 0),
      memoryCount: memories.length
    }
  })

  /**
   * Create a new zone.
   */
  ipcMain.handle('db:zones:create', async (_event, options: { name: string; description?: string; metadata?: Record<string, unknown>; tags?: string[] }): Promise<Zone> => {
    const zoneId = crypto.randomUUID()
    const metadata = metadataToJSON(options.metadata || {})
    const tags = tagsToCypherList(options.tags || [])

    const query = ZoneQueries.create({
      id: zoneId,
      name: options.name,
      description: options.description,
      metadata,
      tags
    })

    const rows = await executeCypher<Record<string, unknown>>(query, 'z agtype')
    return rowToZone(rows[0])
  })

  /**
   * Update a zone.
   */
  ipcMain.handle('db:zones:update', async (_event, zoneId: string, options: { name?: string; description?: string; metadata?: Record<string, unknown>; tags?: string[] }): Promise<Zone> => {
    const now = new Date().toISOString()
    const sets: string[] = [`z.updated_at = '${now}'`]

    if (options.name !== undefined) {
      sets.push(`z.name = '${escapeCypherString(options.name)}'`)
    }
    if (options.description !== undefined) {
      sets.push(`z.description = '${escapeCypherString(options.description)}'`)
    }
    if (options.metadata !== undefined) {
      sets.push(`z.metadata = '${metadataToJSON(options.metadata)}'`)
    }
    if (options.tags !== undefined) {
      sets.push(`z.tags = ${tagsToCypherList(options.tags)}`)
    }

    const query = ZoneQueries.update(zoneId, sets)
    const rows = await executeCypher<Record<string, unknown>>(query, 'z agtype')

    if (rows.length === 0) {
      throw new Error(`Zone not found: ${zoneId}`)
    }

    return rowToZone(rows[0])
  })

  /**
   * Delete a zone and all its contents.
   */
  ipcMain.handle('db:zones:delete', async (_event, zoneId: string): Promise<void> => {
    const query = ZoneQueries.delete(zoneId)
    await executeCypher(query, 'result agtype')
  })

  // --------------------------------------------------------------------------
  // Plan CRUD Handlers (for creating plans in zones)
  // --------------------------------------------------------------------------

  /**
   * Create a new plan in a zone.
   */
  ipcMain.handle('db:plans:create', async (_event, options: { zoneId: string; name: string; description?: string; status?: string; metadata?: Record<string, unknown>; tags?: string[] }): Promise<Plan> => {
    const client = await getClient()

    try {
      await client.query('BEGIN')

      const planId = crypto.randomUUID()
      const metadata = metadataToJSON(options.metadata || {})
      const tags = tagsToCypherList(options.tags || [])

      // Create the plan
      const createQuery = PlanQueries.create({
        id: planId,
        name: options.name,
        description: options.description,
        status: options.status,
        metadata,
        tags
      })
      await executeCypherInTransaction(client, createQuery, 'p agtype')

      // Link plan to zone
      const linkQuery = PlanQueries.linkToZone(planId, options.zoneId)
      const linkRows = await executeCypherInTransaction<Record<string, unknown>>(client, linkQuery, 'p agtype')

      await client.query('COMMIT')

      return rowToPlan(linkRows[0])
    } catch (err) {
      await client.query('ROLLBACK')
      throw err
    } finally {
      client.release()
    }
  })

  /**
   * Update a plan.
   */
  ipcMain.handle('db:plans:update', async (_event, planId: string, options: { name?: string; description?: string; status?: string; metadata?: Record<string, unknown>; tags?: string[] }): Promise<Plan> => {
    const now = new Date().toISOString()
    const sets: string[] = [`p.updated_at = '${now}'`]

    if (options.name !== undefined) {
      sets.push(`p.name = '${escapeCypherString(options.name)}'`)
    }
    if (options.description !== undefined) {
      sets.push(`p.description = '${escapeCypherString(options.description)}'`)
    }
    if (options.status !== undefined) {
      sets.push(`p.status = '${escapeCypherString(options.status)}'`)
    }
    if (options.metadata !== undefined) {
      sets.push(`p.metadata = '${metadataToJSON(options.metadata)}'`)
    }
    if (options.tags !== undefined) {
      sets.push(`p.tags = ${tagsToCypherList(options.tags)}`)
    }

    const query = PlanQueries.update(planId, sets)
    const rows = await executeCypher<Record<string, unknown>>(query, 'p agtype')

    if (rows.length === 0) {
      throw new Error(`Plan not found: ${planId}`)
    }

    return rowToPlan(rows[0])
  })

  /**
   * Delete a plan and all its tasks.
   */
  ipcMain.handle('db:plans:delete', async (_event, planId: string): Promise<void> => {
    const query = PlanQueries.delete(planId)
    await executeCypher(query, 'result agtype')
  })

  /**
   * Move a plan to a different zone.
   */
  ipcMain.handle('db:plans:move', async (_event, planId: string, newZoneId: string): Promise<Plan> => {
    const query = PlanQueries.moveToZone(planId, newZoneId)
    const rows = await executeCypher<Record<string, unknown>>(query, 'p agtype')

    if (rows.length === 0) {
      throw new Error(`Plan not found: ${planId}`)
    }

    return rowToPlan(rows[0])
  })

  // --------------------------------------------------------------------------
  // Memory Handlers
  // --------------------------------------------------------------------------

  /**
   * Create a new memory in a zone.
   */
  ipcMain.handle('db:memories:create', async (_event, options: { zoneId: string; type: string; content: string; metadata?: Record<string, unknown>; tags?: string[] }): Promise<MemoryInZone> => {
    const client = await getClient()

    try {
      await client.query('BEGIN')

      const memoryId = crypto.randomUUID()
      const metadata = metadataToJSON(options.metadata || {})
      const tags = tagsToCypherList(options.tags || [])

      // Create the memory
      const createQuery = MemoryQueries.create({
        id: memoryId,
        type: options.type,
        content: options.content,
        metadata,
        tags
      })
      await executeCypherInTransaction(client, createQuery, 'm agtype')

      // Link memory to zone
      const linkQuery = MemoryQueries.linkToZone(memoryId, options.zoneId)
      const linkRows = await executeCypherInTransaction<Record<string, unknown>>(client, linkQuery, 'm agtype')

      await client.query('COMMIT')

      return rowToMemory(linkRows[0])
    } catch (err) {
      await client.query('ROLLBACK')
      throw err
    } finally {
      client.release()
    }
  })

  /**
   * Update a memory.
   */
  ipcMain.handle('db:memories:update', async (_event, memoryId: string, options: { content?: string; metadata?: Record<string, unknown>; tags?: string[] }): Promise<MemoryInZone> => {
    const now = new Date().toISOString()
    const sets: string[] = [`m.updated_at = '${now}'`]

    if (options.content !== undefined) {
      sets.push(`m.content = '${escapeCypherString(options.content)}'`)
    }
    if (options.metadata !== undefined) {
      sets.push(`m.metadata = '${metadataToJSON(options.metadata)}'`)
    }
    if (options.tags !== undefined) {
      sets.push(`m.tags = ${tagsToCypherList(options.tags)}`)
    }

    const query = MemoryQueries.update(memoryId, sets)
    const rows = await executeCypher<Record<string, unknown>>(query, 'm agtype')

    if (rows.length === 0) {
      throw new Error(`Memory not found: ${memoryId}`)
    }

    return rowToMemory(rows[0])
  })

  /**
   * Delete a memory.
   */
  ipcMain.handle('db:memories:delete', async (_event, memoryId: string): Promise<void> => {
    const query = MemoryQueries.delete(memoryId)
    await executeCypher(query, 'result agtype')
  })

  /**
   * Link a memory to another node via RELATES_TO.
   */
  ipcMain.handle('db:memories:link', async (_event, memoryId: string, targetId: string): Promise<void> => {
    const query = MemoryQueries.linkTo(memoryId, targetId)
    await executeCypher(query, 'm agtype')
  })

  /**
   * Unlink a memory from a node.
   */
  ipcMain.handle('db:memories:unlink', async (_event, memoryId: string, targetId: string): Promise<void> => {
    const query = MemoryQueries.unlinkFrom(memoryId, targetId)
    await executeCypher(query, 'm agtype')
  })
}
