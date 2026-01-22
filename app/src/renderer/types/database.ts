import type { Plan, PlanStatus } from './plan'
import type { Task, TaskInPlan, TaskStatus } from './task'

export interface DatabaseConfig {
  host: string
  port: number
  database: string
  user: string
  password: string
}

export interface ListPlansOptions {
  status?: PlanStatus
  search?: string
  limit?: number
  offset?: number
}

export interface ListTasksOptions {
  planId?: string
  status?: TaskStatus
  search?: string
  limit?: number
  offset?: number
}

export interface CreateTaskOptions {
  planId: string
  content: string
  status?: TaskStatus
  metadata?: Task['metadata']
  tags?: string[]
  position?: number
}

export interface UpdateTaskOptions {
  content?: string
  status?: TaskStatus
  metadata?: Task['metadata']
  tags?: string[]
}

export interface CreateDependencyOptions {
  sourceTaskId: string
  targetTaskId: string
}

export interface PlanWithTasks extends Plan {
  tasks: TaskInPlan[]
}

// IPC Channel types
export interface IpcChannels {
  'db:plans:list': (options?: ListPlansOptions) => Promise<Plan[]>
  'db:plans:get': (planId: string) => Promise<PlanWithTasks | null>
  'db:tasks:create': (options: CreateTaskOptions) => Promise<TaskInPlan>
  'db:tasks:update': (taskId: string, options: UpdateTaskOptions) => Promise<Task>
  'db:tasks:delete': (taskId: string) => Promise<void>
  'db:tasks:reorder': (planId: string, taskIds: string[]) => Promise<void>
  'db:dependencies:create': (options: CreateDependencyOptions) => Promise<void>
  'db:dependencies:delete': (sourceTaskId: string, targetTaskId: string) => Promise<void>
}
