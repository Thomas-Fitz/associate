export type TaskStatus = 'pending' | 'in_progress' | 'completed' | 'cancelled' | 'blocked'

export interface TaskMetadata {
  ui_x?: number
  ui_y?: number
  ui_width?: number
  ui_height?: number
  [key: string]: unknown
}

export interface Task {
  id: string
  content: string
  status: TaskStatus
  metadata: TaskMetadata
  tags: string[]
  createdAt: string
  updatedAt: string
}

export interface TaskInPlan extends Task {
  position: number
  dependsOn: string[]
  blocks: string[]
}

export type RelationshipType = 'DEPENDS_ON' | 'BLOCKS'

export interface EdgeInfo {
  id: string
  sourceTaskId: string
  targetTaskId: string
  relationshipType: RelationshipType
}
