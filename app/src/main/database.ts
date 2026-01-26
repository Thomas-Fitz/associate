import { Pool, PoolClient } from 'pg'

export interface DatabaseConfig {
  host: string
  port: number
  database: string
  user: string
  password: string
}

const defaultConfig: DatabaseConfig = {
  host: process.env.DB_HOST || 'localhost',
  port: parseInt(process.env.DB_PORT || '5432', 10),
  database: process.env.DB_DATABASE || 'associate',
  user: process.env.DB_USERNAME || 'associate',
  password: process.env.DB_PASSWORD || 'password'
}

let pool: Pool | null = null

export function getPool(): Pool {
  if (!pool) {
    pool = new Pool(defaultConfig)
  }
  return pool
}

export async function initializeDatabase(): Promise<void> {
  const client = await getPool().connect()
  try {
    // Initialize AGE extension and graph
    await client.query('CREATE EXTENSION IF NOT EXISTS age;')
    await client.query('SET search_path = ag_catalog, "$user", public;')
    
    // Create the graph if it doesn't exist
    try {
      await client.query(`SELECT * FROM ag_catalog.create_graph('associate');`)
    } catch (err: unknown) {
      // Graph might already exist, which is fine
      const pgErr = err as { code?: string; message?: string }
      // Check for duplicate_object error or "already exists" message
      if (pgErr.code !== '42710' && pgErr.code !== '3F000' && 
          !(pgErr.message && pgErr.message.includes('already exists'))) {
        throw err
      }
      // Graph already exists - this is fine
    }
    
    console.log('Database initialized successfully')
  } finally {
    client.release()
  }
}

export async function closeDatabase(): Promise<void> {
  if (pool) {
    await pool.end()
    pool = null
  }
}

// Helper to escape strings for Cypher queries (port from Go)
export function escapeCypherString(str: string): string {
  return str
    .replace(/\\/g, '\\\\')
    .replace(/'/g, "\\'")
    .replace(/"/g, '\\"')
    .replace(/\n/g, '\\n')
    .replace(/\r/g, '\\r')
    .replace(/\t/g, '\\t')
}

// Helper to convert metadata object to JSON string for Cypher
export function metadataToJSON(metadata: Record<string, unknown>): string {
  return JSON.stringify(metadata).replace(/'/g, "''")
}

// Helper to convert tags array to Cypher list
export function tagsToCypherList(tags: string[]): string {
  if (!tags || tags.length === 0) {
    return '[]'
  }
  const escaped = tags.map(t => `'${escapeCypherString(t)}'`)
  return `[${escaped.join(', ')}]`
}

// Parse AGE vertex/edge result
// AGE returns vertices like: {"id": 12345, "label": "Memory", "properties": {"id": "abc", ...}}::vertex
export function parseAGTypeProperties(result: unknown): Record<string, unknown> {
  console.log('parseAGTypeProperties input:', typeof result, result)
  
  if (!result) return {}
  
  // AGE returns results as strings that need parsing
  if (typeof result === 'string') {
    try {
      // Remove the ::vertex or ::edge suffix
      let jsonStr = result.replace(/::(?:vertex|edge)$/, '').trim()
      
      // If it's empty after trimming, return empty object
      if (!jsonStr) return {}
      
      console.log('Parsing JSON string:', jsonStr)
      const wrapper = JSON.parse(jsonStr) as { properties?: Record<string, unknown> }
      console.log('Parsed wrapper:', wrapper)
      return wrapper.properties || wrapper
    } catch (err) {
      console.error('Failed to parse AGE result:', result, err)
      // Fall through to return empty object
    }
  }
  
  if (typeof result === 'object' && result !== null) {
    const obj = result as Record<string, unknown>
    console.log('Result is object:', obj)
    return obj.properties ? (obj.properties as Record<string, unknown>) : obj
  }
  
  return {}
}

// Execute a Cypher query with AGE
export async function executeCypher<T = unknown>(
  query: string,
  returnColumns: string = 'result agtype'
): Promise<T[]> {
  const client = await getPool().connect()
  try {
    await client.query('SET search_path = ag_catalog, "$user", public;')
    
    const sql = `SELECT * FROM ag_catalog.cypher('associate', $$ ${query} $$) AS (${returnColumns});`
    console.log('Executing SQL:', sql)
    const result = await client.query(sql)
    console.log('Raw result:', JSON.stringify(result.rows, null, 2))
    
    return result.rows as T[]
  } finally {
    client.release()
  }
}

// Execute a Cypher query within a transaction
export async function executeCypherInTransaction<T = unknown>(
  client: PoolClient,
  query: string,
  returnColumns: string = 'result agtype'
): Promise<T[]> {
  await client.query('SET search_path = ag_catalog, "$user", public;')
  
  const sql = `SELECT * FROM ag_catalog.cypher('associate', $$ ${query} $$) AS (${returnColumns});`
  const result = await client.query(sql)
  
  return result.rows as T[]
}

// Get a client for transaction operations
export async function getClient(): Promise<PoolClient> {
  return getPool().connect()
}
