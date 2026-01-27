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

// Helper to parse metadata safely
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

// Helper to convert AGE result to Plan
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

// Helper to convert AGE result to Task
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

// Helper to parse AGE array result (can be string or array)
function parseAGArray(value: unknown): string[] {
  if (!value) return []
  if (Array.isArray(value)) {
    return value.map(v => String(v)).filter(v => v && v !== 'null')
  }
  if (typeof value === 'string') {
    // AGE returns arrays as strings like: ["id1", "id2"] or [null]
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

// Helper to convert AGE result to TaskInPlan
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
  
  console.log('rowToTaskInPlan:', { taskId: task.id, position, dependsOn, blocks, rawDepsOn: row.depends_on, rawBlocks: row.blocks })
  
  return {
    ...task,
    position,
    dependsOn,
    blocks
  }
}

export function setupIpcHandlers(): void {
  // List plans with optional filtering
  ipcMain.handle('db:plans:list', async (_event, options?: ListPlansOptions) => {
    console.log('db:plans:list called with options:', options)
    
    let query = 'MATCH (p:Plan)'
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
    
    // Get task count for each plan
    query += `
      OPTIONAL MATCH (p)-[:CONTAINS]->(t:Task)
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
    
    console.log('Executing query:', query)
    const rows = await executeCypher<Record<string, unknown>>(query, 'p agtype, task_count agtype')
    console.log('Query returned rows:', rows.length, rows)
    
    const plans = rows.map(row => {
      console.log('Processing row:', row)
      const plan = rowToPlan(row)
      plan.taskCount = typeof row.task_count === 'number' ? row.task_count : 
        (typeof row.task_count === 'string' ? parseInt(row.task_count, 10) : 0)
      console.log('Converted to plan:', plan)
      return plan
    })
    
    console.log('Returning plans:', plans)
    return plans
  })

  // Get a single plan with its tasks
  ipcMain.handle('db:plans:get', async (_event, planId: string): Promise<PlanWithTasks | null> => {
    const escapedId = escapeCypherString(planId)
    
    // Get the plan
    const planQuery = `MATCH (p:Plan {id: '${escapedId}'}) RETURN p`
    const planRows = await executeCypher<Record<string, unknown>>(planQuery, 'p agtype')
    
    if (planRows.length === 0) {
      return null
    }
    
    const plan = rowToPlan(planRows[0])
    
    // Get tasks with their positions and relationships
    const tasksQuery = `
      MATCH (p:Plan {id: '${escapedId}'})-[r:CONTAINS]->(t:Task)
      OPTIONAL MATCH (t)-[:DEPENDS_ON]->(dep:Task)
      OPTIONAL MATCH (t)-[:BLOCKS]->(blk:Task)
      WITH t, r.position as position, collect(DISTINCT dep.id) as depends_on, collect(DISTINCT blk.id) as blocks
      RETURN t, position, depends_on, blocks
      ORDER BY position
    `
    
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

  // Create a new task
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
        // Get the max position in the plan
        const maxPosQuery = `
          MATCH (p:Plan {id: '${escapeCypherString(options.planId)}'})-[r:CONTAINS]->(t:Task)
          RETURN max(r.position) as max_pos
        `
        const posRows = await executeCypherInTransaction<{ max_pos: unknown }>(
          client, 
          maxPosQuery,
          'max_pos agtype'
        )
        const maxPos = posRows[0]?.max_pos
        position = (typeof maxPos === 'number' ? maxPos : 0) + 1000
      }
      
      // Create the task
      const createQuery = `
        CREATE (t:Task {
          id: '${taskId}',
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
      
      // Link task to plan
      const linkQuery = `
        MATCH (p:Plan {id: '${escapeCypherString(options.planId)}'}), (t:Task {id: '${taskId}'})
        CREATE (p)-[:CONTAINS {position: ${position}}]->(t)
        RETURN t
      `
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

  // Update a task
  ipcMain.handle('db:tasks:update', async (_event, taskId: string, options: UpdateTaskOptions): Promise<Task> => {
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
      MATCH (t:Task {id: '${escapedId}'})
      SET ${sets.join(', ')}
      RETURN t
    `
    
    const rows = await executeCypher<Record<string, unknown>>(query, 't agtype')
    
    if (rows.length === 0) {
      throw new Error(`Task not found: ${taskId}`)
    }
    
    return rowToTask(rows[0])
  })

  // Delete a task
  ipcMain.handle('db:tasks:delete', async (_event, taskId: string): Promise<void> => {
    const escapedId = escapeCypherString(taskId)
    
    // Delete relationships first, then the task
    const query = `
      MATCH (t:Task {id: '${escapedId}'})
      DETACH DELETE t
    `
    
    await executeCypher(query, 'result agtype')
  })

  // Reorder tasks within a plan
  ipcMain.handle('db:tasks:reorder', async (_event, planId: string, taskIds: string[]): Promise<void> => {
    const client = await getClient()
    
    try {
      await client.query('BEGIN')
      
      const escapedPlanId = escapeCypherString(planId)
      
      // Update each task's position
      for (let i = 0; i < taskIds.length; i++) {
        const position = (i + 1) * 1000 // Use 1000, 2000, 3000... for easy insertion
        const escapedTaskId = escapeCypherString(taskIds[i])
        
        const query = `
          MATCH (p:Plan {id: '${escapedPlanId}'})-[r:CONTAINS]->(t:Task {id: '${escapedTaskId}'})
          SET r.position = ${position}
          RETURN t
        `
        await executeCypherInTransaction(client, query, 't agtype')
      }
      
      await client.query('COMMIT')
    } catch (err) {
      await client.query('ROLLBACK')
      throw err
    } finally {
      client.release()
    }
  })

  // Create a dependency between tasks
  ipcMain.handle('db:dependencies:create', async (_event, sourceTaskId: string, targetTaskId: string): Promise<void> => {
    const escapedSource = escapeCypherString(sourceTaskId)
    const escapedTarget = escapeCypherString(targetTaskId)
    
    // Check for circular dependency
    const circularCheck = `
      MATCH path = (target:Task {id: '${escapedTarget}'})-[:DEPENDS_ON*]->(source:Task {id: '${escapedSource}'})
      RETURN count(path) as cycle_count
    `
    
    const checkRows = await executeCypher<{ cycle_count: unknown }>(circularCheck, 'cycle_count agtype')
    const cycleCount = checkRows[0]?.cycle_count
    if (typeof cycleCount === 'number' && cycleCount > 0) {
      throw new Error('Cannot create circular dependency')
    }
    
    // Create the dependency
    const query = `
      MATCH (source:Task {id: '${escapedSource}'}), (target:Task {id: '${escapedTarget}'})
      MERGE (source)-[:DEPENDS_ON]->(target)
      RETURN source
    `
    
    await executeCypher(query, 'source agtype')
  })

  // Delete a dependency between tasks
  ipcMain.handle('db:dependencies:delete', async (_event, sourceTaskId: string, targetTaskId: string, relationshipType: 'DEPENDS_ON' | 'BLOCKS' = 'DEPENDS_ON'): Promise<void> => {
    const escapedSource = escapeCypherString(sourceTaskId)
    const escapedTarget = escapeCypherString(targetTaskId)
    
    const query = `
      MATCH (source:Task {id: '${escapedSource}'})-[r:${relationshipType}]->(target:Task {id: '${escapedTarget}'})
      DELETE r
      RETURN source
    `
    
    await executeCypher(query, 'source agtype')
  })
}
