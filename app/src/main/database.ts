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
      const pgErr = err as { code?: string }
      if (pgErr.code !== '42710') { // duplicate_object
        throw err
      }
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
export function parseAGTypeProperties(result: unknown): Record<string, unknown> {
  if (!result) return {}
  
  // AGE returns results as strings that need parsing
  if (typeof result === 'string') {
    try {
      // AGE format: {id: ..., label: "...", properties: {...}}::vertex
      const match = result.match(/\{.*\}/)
      if (match) {
        // Parse the JSON-like structure
        let parsed = match[0]
          .replace(/(\w+):/g, '"$1":') // Add quotes around keys
          .replace(/::vertex$/, '')
          .replace(/::edge$/, '')
        
        const obj = JSON.parse(parsed)
        return obj.properties || obj
      }
    } catch {
      // Fall through to return empty object
    }
  }
  
  if (typeof result === 'object' && result !== null) {
    const obj = result as Record<string, unknown>
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
    const result = await client.query(sql)
    
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
