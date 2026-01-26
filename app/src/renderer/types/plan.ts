export type PlanStatus = 'draft' | 'active' | 'completed' | 'archived'

export interface PlanMetadata {
  [key: string]: unknown
}

export interface Plan {
  id: string
  name: string
  description: string
  status: PlanStatus
  metadata: PlanMetadata
  tags: string[]
  createdAt: string
  updatedAt: string
  taskCount?: number
}
