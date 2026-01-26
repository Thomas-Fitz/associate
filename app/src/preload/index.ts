import { contextBridge, ipcRenderer } from 'electron'
import type {
  Plan,
  Task,
  TaskInPlan,
  ListPlansOptions,
  CreateTaskOptions,
  UpdateTaskOptions,
  PlanWithTasks
} from '../renderer/types'

// Expose database operations to renderer
const api = {
  // Plan operations
  plans: {
    list: (options?: ListPlansOptions): Promise<Plan[]> => 
      ipcRenderer.invoke('db:plans:list', options),
    get: (planId: string): Promise<PlanWithTasks | null> => 
      ipcRenderer.invoke('db:plans:get', planId)
  },
  
  // Task operations
  tasks: {
    create: (options: CreateTaskOptions): Promise<TaskInPlan> => 
      ipcRenderer.invoke('db:tasks:create', options),
    update: (taskId: string, options: UpdateTaskOptions): Promise<Task> => 
      ipcRenderer.invoke('db:tasks:update', taskId, options),
    delete: (taskId: string): Promise<void> => 
      ipcRenderer.invoke('db:tasks:delete', taskId),
    reorder: (planId: string, taskIds: string[]): Promise<void> => 
      ipcRenderer.invoke('db:tasks:reorder', planId, taskIds)
  },
  
  // Dependency operations
  dependencies: {
    create: (sourceTaskId: string, targetTaskId: string): Promise<void> => 
      ipcRenderer.invoke('db:dependencies:create', sourceTaskId, targetTaskId),
    delete: (sourceTaskId: string, targetTaskId: string): Promise<void> => 
      ipcRenderer.invoke('db:dependencies:delete', sourceTaskId, targetTaskId)
  }
}

// Type for the exposed API
export type ElectronAPI = typeof api

// Expose the API to the renderer
contextBridge.exposeInMainWorld('electronAPI', api)
