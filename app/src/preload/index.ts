import { contextBridge, ipcRenderer, IpcRendererEvent } from 'electron'
import type {
  Plan,
  Task,
  TaskInPlan,
  ListPlansOptions,
  CreateTaskOptions,
  UpdateTaskOptions,
  PlanWithTasks,
  Zone,
  ZoneWithContents,
  MemoryInZone,
  TerminalInZone,
  TerminalConfig,
  TerminalCreateOptions,
  TerminalUpdateOptions,
  PtyDataEvent,
  PtyExitEvent
} from '../renderer/types'

// Expose database operations to renderer
const api = {
  // Zone operations
  zones: {
    list: (options?: { search?: string; limit?: number }): Promise<Zone[]> =>
      ipcRenderer.invoke('db:zones:list', options),
    get: (zoneId: string): Promise<ZoneWithContents | null> =>
      ipcRenderer.invoke('db:zones:get', zoneId),
    getById: (zoneId: string): Promise<Zone | null> =>
      ipcRenderer.invoke('db:zones:getById', zoneId),
    create: (options: { name: string; description?: string; metadata?: Record<string, unknown>; tags?: string[] }): Promise<Zone> =>
      ipcRenderer.invoke('db:zones:create', options),
    update: (zoneId: string, options: { name?: string; description?: string; metadata?: Record<string, unknown>; tags?: string[] }): Promise<Zone> =>
      ipcRenderer.invoke('db:zones:update', zoneId, options),
    delete: (zoneId: string): Promise<void> =>
      ipcRenderer.invoke('db:zones:delete', zoneId)
  },

  // Plan operations
  plans: {
    list: (options?: ListPlansOptions): Promise<Plan[]> => 
      ipcRenderer.invoke('db:plans:list', options),
    get: (planId: string): Promise<PlanWithTasks | null> => 
      ipcRenderer.invoke('db:plans:get', planId),
    create: (options: { zoneId: string; name: string; description?: string; status?: string; metadata?: Record<string, unknown>; tags?: string[] }): Promise<Plan> =>
      ipcRenderer.invoke('db:plans:create', options),
    update: (planId: string, options: { name?: string; description?: string; status?: string; metadata?: Record<string, unknown>; tags?: string[] }): Promise<Plan> =>
      ipcRenderer.invoke('db:plans:update', planId, options),
    delete: (planId: string): Promise<void> =>
      ipcRenderer.invoke('db:plans:delete', planId),
    move: (planId: string, newZoneId: string): Promise<Plan> =>
      ipcRenderer.invoke('db:plans:move', planId, newZoneId)
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
      ipcRenderer.invoke('db:tasks:reorder', planId, taskIds),
    move: (taskId: string, newPlanId: string, options?: { position?: number; metadata?: Record<string, unknown> }): Promise<Task> =>
      ipcRenderer.invoke('db:tasks:move', taskId, newPlanId, options)
  },
  
  // Dependency operations
  dependencies: {
    create: (sourceTaskId: string, targetTaskId: string): Promise<void> => 
      ipcRenderer.invoke('db:dependencies:create', sourceTaskId, targetTaskId),
    delete: (sourceTaskId: string, targetTaskId: string, relationshipType: 'DEPENDS_ON' | 'BLOCKS' = 'DEPENDS_ON'): Promise<void> => 
      ipcRenderer.invoke('db:dependencies:delete', sourceTaskId, targetTaskId, relationshipType)
  },

  // Memory operations
  memories: {
    create: (options: { zoneId: string; type: string; content: string; metadata?: Record<string, unknown>; tags?: string[] }): Promise<MemoryInZone> =>
      ipcRenderer.invoke('db:memories:create', options),
    update: (memoryId: string, options: { content?: string; metadata?: Record<string, unknown>; tags?: string[] }): Promise<MemoryInZone> =>
      ipcRenderer.invoke('db:memories:update', memoryId, options),
    delete: (memoryId: string): Promise<void> =>
      ipcRenderer.invoke('db:memories:delete', memoryId),
    linkTo: (memoryId: string, targetId: string): Promise<void> =>
      ipcRenderer.invoke('db:memories:link', memoryId, targetId),
    unlinkFrom: (memoryId: string, targetId: string): Promise<void> =>
      ipcRenderer.invoke('db:memories:unlink', memoryId, targetId)
  },

  // Terminal DB operations (for persistence)
  terminals: {
    list: (zoneId: string): Promise<TerminalInZone[]> =>
      ipcRenderer.invoke('db:terminals:list', zoneId),
    create: (options: TerminalCreateOptions): Promise<TerminalInZone> =>
      ipcRenderer.invoke('db:terminals:create', options),
    update: (terminalId: string, options: TerminalUpdateOptions): Promise<TerminalInZone> =>
      ipcRenderer.invoke('db:terminals:update', terminalId, options),
    delete: (terminalId: string): Promise<void> =>
      ipcRenderer.invoke('db:terminals:delete', terminalId)
  },

  // PTY operations (for live terminal interaction)
  pty: {
    create: (terminalId: string, config: TerminalConfig): Promise<void> =>
      ipcRenderer.invoke('pty:create', terminalId, config),
    write: (terminalId: string, data: string): Promise<void> =>
      ipcRenderer.invoke('pty:write', terminalId, data),
    resize: (terminalId: string, cols: number, rows: number): Promise<void> =>
      ipcRenderer.invoke('pty:resize', terminalId, cols, rows),
    kill: (terminalId: string): Promise<void> =>
      ipcRenderer.invoke('pty:kill', terminalId),
    loadScrollback: (terminalId: string): Promise<string> =>
      ipcRenderer.invoke('pty:loadScrollback', terminalId),
    isRunning: (terminalId: string): Promise<boolean> =>
      ipcRenderer.invoke('pty:isRunning', terminalId),
    getRunningCount: (): Promise<number> =>
      ipcRenderer.invoke('pty:getRunningCount'),
    
    // Event subscriptions (returns unsubscribe function)
    onData: (callback: (data: PtyDataEvent) => void) => {
      const handler = (_event: IpcRendererEvent, data: PtyDataEvent) => callback(data)
      ipcRenderer.on('pty:data', handler)
      return () => ipcRenderer.removeListener('pty:data', handler)
    },
    onExit: (callback: (data: PtyExitEvent) => void) => {
      const handler = (_event: IpcRendererEvent, data: PtyExitEvent) => callback(data)
      ipcRenderer.on('pty:exit', handler)
      return () => ipcRenderer.removeListener('pty:exit', handler)
    }
  }
}

// Type for the exposed API
export type ElectronAPI = typeof api

// Expose the API to the renderer
contextBridge.exposeInMainWorld('electronAPI', api)
