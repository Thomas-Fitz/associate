// Zone types for Phase 0 prototype

import type { TerminalInZone } from './terminal'

export interface ZoneMetadata {
  [key: string]: unknown
}

export interface Zone {
  id: string
  name: string
  description: string
  metadata: ZoneMetadata
  tags: string[]
  createdAt: string
  updatedAt: string
  planCount?: number
  taskCount?: number
  memoryCount?: number
  terminalCount?: number
}

export interface MemoryInZone {
  id: string
  type: 'Note' | 'Repository' | 'Memory'
  content: string
  metadata: Record<string, unknown>
  tags: string[]
  createdAt: string
  updatedAt: string
  // UI positioning (stored in metadata on the node)
  ui_x?: number
  ui_y?: number
  ui_width?: number
  ui_height?: number
}

export interface TaskInZone {
  id: string
  content: string
  status: 'pending' | 'in_progress' | 'completed' | 'cancelled' | 'blocked'
  metadata: {
    ui_x?: number
    ui_y?: number
    ui_width?: number
    ui_height?: number
    [key: string]: unknown
  }
  tags: string[]
  createdAt: string
  updatedAt: string
  planId: string  // Which plan this task belongs to
  dependsOn: string[]
  blocks: string[]
}

export interface PlanInZone {
  id: string
  name: string
  description: string
  status: 'draft' | 'active' | 'completed' | 'archived'
  metadata: {
    ui_x?: number
    ui_y?: number
    ui_width?: number
    ui_height?: number
    [key: string]: unknown
  }
  tags: string[]
  createdAt: string
  updatedAt: string
  tasks: TaskInZone[]
}

export interface ZoneWithContents extends Zone {
  plans: PlanInZone[]
  memories: MemoryInZone[]
  terminals: TerminalInZone[]
}
