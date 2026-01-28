import type { ElectronAPI } from '../../preload'
import type { 
  Plan, 
  TaskInPlan, 
  PlanWithTasks, 
  ListPlansOptions, 
  CreateTaskOptions, 
  UpdateTaskOptions,
  Zone,
  ZoneWithContents,
  PlanInZone,
  TaskInZone,
  MemoryInZone
} from '../types'

declare global {
  interface Window {
    electronAPI: ElectronAPI
  }
}

// Mock data for browser testing
const mockZones: Zone[] = [
  {
    id: 'zone-1',
    name: 'Product Development',
    description: 'All product development work',
    metadata: {},
    tags: ['product'],
    createdAt: '2024-01-01T10:00:00Z',
    updatedAt: '2024-01-25T15:30:00Z',
    planCount: 2,
    taskCount: 5,
    memoryCount: 1
  },
  {
    id: 'zone-2',
    name: 'Marketing',
    description: 'Marketing campaigns and initiatives',
    metadata: {},
    tags: ['marketing'],
    createdAt: '2024-01-05T08:00:00Z',
    updatedAt: '2024-01-20T12:00:00Z',
    planCount: 1,
    taskCount: 2,
    memoryCount: 0
  }
]

const mockPlans: Plan[] = [
  {
    id: 'plan-1',
    name: 'Website Redesign',
    description: 'Redesign the company website with modern UI',
    status: 'active',
    metadata: { ui_x: 50, ui_y: 50, ui_width: 400, ui_height: 300 },
    tags: ['design', 'frontend'],
    createdAt: '2024-01-15T10:00:00Z',
    updatedAt: '2024-01-20T15:30:00Z',
    taskCount: 3
  },
  {
    id: 'plan-2',
    name: 'API Integration',
    description: 'Integrate third-party payment API',
    status: 'draft',
    metadata: { ui_x: 500, ui_y: 50, ui_width: 400, ui_height: 250 },
    tags: ['backend', 'api'],
    createdAt: '2024-01-10T08:00:00Z',
    updatedAt: '2024-01-18T12:00:00Z',
    taskCount: 2
  },
  {
    id: 'plan-3',
    name: 'Mobile App Launch',
    description: 'Launch mobile app on iOS and Android',
    status: 'completed',
    metadata: { ui_x: 50, ui_y: 50, ui_width: 400, ui_height: 300 },
    tags: ['mobile', 'launch'],
    createdAt: '2024-01-01T09:00:00Z',
    updatedAt: '2024-01-25T18:00:00Z',
    taskCount: 2
  }
]

const mockTasks: Record<string, TaskInPlan[]> = {
  'plan-1': [
    {
      id: 'task-1-1',
      content: 'Create wireframes for homepage',
      status: 'completed',
      metadata: { ui_x: 50, ui_y: 80, ui_width: 250, ui_height: 80 },
      tags: ['design'],
      createdAt: '2024-01-15T10:00:00Z',
      updatedAt: '2024-01-16T14:00:00Z',
      position: 1000,
      dependsOn: [],
      blocks: []
    },
    {
      id: 'task-1-2',
      content: 'Design UI components in Figma',
      status: 'in_progress',
      metadata: { ui_x: 50, ui_y: 180, ui_width: 250, ui_height: 80 },
      tags: ['design'],
      createdAt: '2024-01-16T09:00:00Z',
      updatedAt: '2024-01-20T11:00:00Z',
      position: 2000,
      dependsOn: ['task-1-1'],
      blocks: []
    },
    {
      id: 'task-1-3',
      content: 'Implement responsive layout',
      status: 'pending',
      metadata: { ui_x: 50, ui_y: 280, ui_width: 250, ui_height: 80 },
      tags: ['frontend'],
      createdAt: '2024-01-17T08:00:00Z',
      updatedAt: '2024-01-17T08:00:00Z',
      position: 3000,
      dependsOn: ['task-1-2'],
      blocks: []
    }
  ],
  'plan-2': [
    {
      id: 'task-2-1',
      content: 'Research payment API documentation',
      status: 'pending',
      metadata: { ui_x: 50, ui_y: 80, ui_width: 250, ui_height: 80 },
      tags: ['research'],
      createdAt: '2024-01-10T08:00:00Z',
      updatedAt: '2024-01-10T08:00:00Z',
      position: 1000,
      dependsOn: [],
      blocks: []
    },
    {
      id: 'task-2-2',
      content: 'Implement payment endpoints',
      status: 'blocked',
      metadata: { ui_x: 50, ui_y: 180, ui_width: 250, ui_height: 80 },
      tags: ['backend'],
      createdAt: '2024-01-11T09:00:00Z',
      updatedAt: '2024-01-11T09:00:00Z',
      position: 2000,
      dependsOn: ['task-2-1'],
      blocks: []
    }
  ],
  'plan-3': [
    {
      id: 'task-3-1',
      content: 'Submit app to App Store',
      status: 'completed',
      metadata: { ui_x: 50, ui_y: 80, ui_width: 250, ui_height: 80 },
      tags: ['ios'],
      createdAt: '2024-01-01T09:00:00Z',
      updatedAt: '2024-01-05T16:00:00Z',
      position: 1000,
      dependsOn: [],
      blocks: []
    },
    {
      id: 'task-3-2',
      content: 'Submit app to Play Store',
      status: 'completed',
      metadata: { ui_x: 50, ui_y: 180, ui_width: 250, ui_height: 80 },
      tags: ['android'],
      createdAt: '2024-01-01T10:00:00Z',
      updatedAt: '2024-01-06T14:00:00Z',
      position: 2000,
      dependsOn: [],
      blocks: []
    }
  ]
}

const mockMemories: Record<string, MemoryInZone[]> = {
  'zone-1': [
    {
      id: 'memory-1',
      type: 'Note',
      content: 'Remember to coordinate with design team on the new color palette',
      metadata: { ui_x: 950, ui_y: 100 },
      tags: ['reminder'],
      createdAt: '2024-01-15T10:00:00Z',
      updatedAt: '2024-01-15T10:00:00Z',
      ui_x: 950,
      ui_y: 100
    }
  ],
  'zone-2': []
}

// Zone to plans mapping
const zonePlanMapping: Record<string, string[]> = {
  'zone-1': ['plan-1', 'plan-2'],
  'zone-2': ['plan-3']
}

// In-memory state for mock mode
let mockState = {
  zones: [...mockZones],
  plans: [...mockPlans],
  tasks: JSON.parse(JSON.stringify(mockTasks)) as Record<string, TaskInPlan[]>,
  memories: JSON.parse(JSON.stringify(mockMemories)) as Record<string, MemoryInZone[]>,
  zonePlanMapping: { ...zonePlanMapping }
}

function createMockAPI(): ElectronAPI {
  return {
    zones: {
      list: async (options?: { search?: string; limit?: number }): Promise<Zone[]> => {
        let result = [...mockState.zones]
        
        if (options?.search) {
          const search = options.search.toLowerCase()
          result = result.filter(z => 
            z.name.toLowerCase().includes(search) || 
            z.description.toLowerCase().includes(search)
          )
        }
        
        if (options?.limit) {
          result = result.slice(0, options.limit)
        }
        
        return result
      },

      get: async (zoneId: string): Promise<ZoneWithContents | null> => {
        const zone = mockState.zones.find(z => z.id === zoneId)
        if (!zone) return null

        const planIds = mockState.zonePlanMapping[zoneId] || []
        const plans: PlanInZone[] = planIds.map(planId => {
          const plan = mockState.plans.find(p => p.id === planId)
          if (!plan) return null

          const tasks: TaskInZone[] = (mockState.tasks[planId] || []).map(t => ({
            id: t.id,
            content: t.content,
            status: t.status as TaskInZone['status'],
            metadata: t.metadata as TaskInZone['metadata'],
            tags: t.tags,
            createdAt: t.createdAt,
            updatedAt: t.updatedAt,
            planId: planId,
            dependsOn: t.dependsOn,
            blocks: t.blocks
          }))

          return {
            id: plan.id,
            name: plan.name,
            description: plan.description,
            status: plan.status as PlanInZone['status'],
            metadata: plan.metadata as PlanInZone['metadata'],
            tags: plan.tags,
            createdAt: plan.createdAt,
            updatedAt: plan.updatedAt,
            tasks
          }
        }).filter((p): p is PlanInZone => p !== null)

        const memories = mockState.memories[zoneId] || []

        return {
          ...zone,
          plans,
          memories,
          planCount: plans.length,
          taskCount: plans.reduce((sum, p) => sum + p.tasks.length, 0),
          memoryCount: memories.length
        }
      },

      getById: async (zoneId: string): Promise<Zone | null> => {
        return mockState.zones.find(z => z.id === zoneId) || null
      },

      create: async (options: { name: string; description?: string; metadata?: Record<string, unknown>; tags?: string[] }): Promise<Zone> => {
        const now = new Date().toISOString()
        const newZone: Zone = {
          id: `zone-${Date.now()}`,
          name: options.name,
          description: options.description || '',
          metadata: options.metadata || {},
          tags: options.tags || [],
          createdAt: now,
          updatedAt: now,
          planCount: 0,
          taskCount: 0,
          memoryCount: 0
        }
        mockState.zones.push(newZone)
        mockState.zonePlanMapping[newZone.id] = []
        mockState.memories[newZone.id] = []
        return newZone
      },

      update: async (zoneId: string, options: { name?: string; description?: string; metadata?: Record<string, unknown>; tags?: string[] }): Promise<Zone> => {
        const zone = mockState.zones.find(z => z.id === zoneId)
        if (!zone) throw new Error(`Zone not found: ${zoneId}`)
        
        if (options.name !== undefined) zone.name = options.name
        if (options.description !== undefined) zone.description = options.description
        if (options.metadata !== undefined) zone.metadata = options.metadata
        if (options.tags !== undefined) zone.tags = options.tags
        zone.updatedAt = new Date().toISOString()
        
        return zone
      },

      delete: async (zoneId: string): Promise<void> => {
        const planIds = mockState.zonePlanMapping[zoneId] || []
        planIds.forEach(planId => {
          delete mockState.tasks[planId]
          mockState.plans = mockState.plans.filter(p => p.id !== planId)
        })
        delete mockState.zonePlanMapping[zoneId]
        delete mockState.memories[zoneId]
        mockState.zones = mockState.zones.filter(z => z.id !== zoneId)
      }
    },

    plans: {
      list: async (options?: ListPlansOptions): Promise<Plan[]> => {
        let result = [...mockState.plans]
        
        if (options?.status && (options.status as string) !== 'all') {
          result = result.filter(p => p.status === options.status)
        }
        
        if (options?.search) {
          const search = options.search.toLowerCase()
          result = result.filter(p => 
            p.name.toLowerCase().includes(search) || 
            p.description.toLowerCase().includes(search)
          )
        }
        
        return result
      },
      
      get: async (planId: string): Promise<PlanWithTasks | null> => {
        const plan = mockState.plans.find(p => p.id === planId)
        if (!plan) return null
        
        return {
          ...plan,
          tasks: mockState.tasks[planId] || []
        }
      },

      create: async (options: { zoneId: string; name: string; description?: string; status?: string; metadata?: Record<string, unknown>; tags?: string[] }): Promise<Plan> => {
        const now = new Date().toISOString()
        const newPlan: Plan = {
          id: `plan-${Date.now()}`,
          name: options.name,
          description: options.description || '',
          status: (options.status as Plan['status']) || 'draft',
          metadata: options.metadata || {},
          tags: options.tags || [],
          createdAt: now,
          updatedAt: now,
          taskCount: 0
        }
        mockState.plans.push(newPlan)
        mockState.tasks[newPlan.id] = []
        if (!mockState.zonePlanMapping[options.zoneId]) {
          mockState.zonePlanMapping[options.zoneId] = []
        }
        mockState.zonePlanMapping[options.zoneId].push(newPlan.id)
        return newPlan
      },

      update: async (planId: string, options: { name?: string; description?: string; status?: string; metadata?: Record<string, unknown>; tags?: string[] }): Promise<Plan> => {
        const plan = mockState.plans.find(p => p.id === planId)
        if (!plan) throw new Error(`Plan not found: ${planId}`)
        
        if (options.name !== undefined) plan.name = options.name
        if (options.description !== undefined) plan.description = options.description
        if (options.status !== undefined) plan.status = options.status as Plan['status']
        if (options.metadata !== undefined) plan.metadata = options.metadata
        if (options.tags !== undefined) plan.tags = options.tags
        plan.updatedAt = new Date().toISOString()
        
        return plan
      },

      delete: async (planId: string): Promise<void> => {
        delete mockState.tasks[planId]
        mockState.plans = mockState.plans.filter(p => p.id !== planId)
        // Remove from zone mapping
        for (const zoneId in mockState.zonePlanMapping) {
          mockState.zonePlanMapping[zoneId] = mockState.zonePlanMapping[zoneId].filter(id => id !== planId)
        }
      },

      move: async (planId: string, newZoneId: string): Promise<Plan> => {
        const plan = mockState.plans.find(p => p.id === planId)
        if (!plan) throw new Error(`Plan not found: ${planId}`)
        
        // Remove from old zone
        for (const zoneId in mockState.zonePlanMapping) {
          mockState.zonePlanMapping[zoneId] = mockState.zonePlanMapping[zoneId].filter(id => id !== planId)
        }
        
        // Add to new zone
        if (!mockState.zonePlanMapping[newZoneId]) {
          mockState.zonePlanMapping[newZoneId] = []
        }
        mockState.zonePlanMapping[newZoneId].push(planId)
        
        return plan
      }
    },
    
    tasks: {
      create: async (options: CreateTaskOptions): Promise<TaskInPlan> => {
        const taskId = `task-${Date.now()}`
        const now = new Date().toISOString()
        
        const planTasks = mockState.tasks[options.planId] || []
        const maxPosition = planTasks.reduce((max, t) => Math.max(max, t.position), 0)
        
        const newTask: TaskInPlan = {
          id: taskId,
          content: options.content,
          status: options.status || 'pending',
          metadata: options.metadata || {},
          tags: options.tags || [],
          createdAt: now,
          updatedAt: now,
          position: options.position ?? maxPosition + 1000,
          dependsOn: [],
          blocks: []
        }
        
        if (!mockState.tasks[options.planId]) {
          mockState.tasks[options.planId] = []
        }
        mockState.tasks[options.planId].push(newTask)
        
        // Update task count
        const plan = mockState.plans.find(p => p.id === options.planId)
        if (plan) {
          plan.taskCount = (plan.taskCount || 0) + 1
        }
        
        return newTask
      },
      
      update: async (taskId: string, options: UpdateTaskOptions) => {
        for (const planId in mockState.tasks) {
          const taskIndex = mockState.tasks[planId].findIndex(t => t.id === taskId)
          if (taskIndex !== -1) {
            const task = mockState.tasks[planId][taskIndex]
            const updatedTask = {
              ...task,
              ...options,
              metadata: options.metadata ?? task.metadata,
              updatedAt: new Date().toISOString()
            }
            mockState.tasks[planId][taskIndex] = updatedTask
            return updatedTask
          }
        }
        throw new Error(`Task not found: ${taskId}`)
      },
      
      delete: async (taskId: string): Promise<void> => {
        for (const planId in mockState.tasks) {
          const taskIndex = mockState.tasks[planId].findIndex(t => t.id === taskId)
          if (taskIndex !== -1) {
            mockState.tasks[planId].splice(taskIndex, 1)
            
            // Update task count
            const plan = mockState.plans.find(p => p.id === planId)
            if (plan && plan.taskCount) {
              plan.taskCount--
            }
            return
          }
        }
      },
      
      reorder: async (planId: string, taskIds: string[]): Promise<void> => {
        const tasks = mockState.tasks[planId]
        if (!tasks) return
        
        taskIds.forEach((taskId, index) => {
          const task = tasks.find(t => t.id === taskId)
          if (task) {
            task.position = (index + 1) * 1000
          }
        })
      }
    },
    
    dependencies: {
      create: async (sourceTaskId: string, targetTaskId: string): Promise<void> => {
        for (const planId in mockState.tasks) {
          const sourceTask = mockState.tasks[planId].find(t => t.id === sourceTaskId)
          if (sourceTask && !sourceTask.dependsOn.includes(targetTaskId)) {
            sourceTask.dependsOn.push(targetTaskId)
            return
          }
        }
      },
      
      delete: async (sourceTaskId: string, targetTaskId: string, relationshipType: 'DEPENDS_ON' | 'BLOCKS' = 'DEPENDS_ON'): Promise<void> => {
        for (const planId in mockState.tasks) {
          const sourceTask = mockState.tasks[planId].find(t => t.id === sourceTaskId)
          if (sourceTask) {
            if (relationshipType === 'DEPENDS_ON') {
              sourceTask.dependsOn = sourceTask.dependsOn.filter(id => id !== targetTaskId)
            } else {
              sourceTask.blocks = sourceTask.blocks.filter(id => id !== targetTaskId)
            }
            return
          }
        }
      }
    },

    memories: {
      create: async (options: { zoneId: string; type: string; content: string; metadata?: Record<string, unknown>; tags?: string[] }): Promise<MemoryInZone> => {
        const now = new Date().toISOString()
        const metadata = options.metadata || {}
        const newMemory: MemoryInZone = {
          id: `memory-${Date.now()}`,
          type: options.type as MemoryInZone['type'],
          content: options.content,
          metadata,
          tags: options.tags || [],
          createdAt: now,
          updatedAt: now,
          ui_x: typeof metadata.ui_x === 'number' ? metadata.ui_x : undefined,
          ui_y: typeof metadata.ui_y === 'number' ? metadata.ui_y : undefined
        }
        
        if (!mockState.memories[options.zoneId]) {
          mockState.memories[options.zoneId] = []
        }
        mockState.memories[options.zoneId].push(newMemory)
        
        return newMemory
      },

      update: async (memoryId: string, options: { content?: string; metadata?: Record<string, unknown>; tags?: string[] }): Promise<MemoryInZone> => {
        for (const zoneId in mockState.memories) {
          const memoryIndex = mockState.memories[zoneId].findIndex(m => m.id === memoryId)
          if (memoryIndex !== -1) {
            const memory = mockState.memories[zoneId][memoryIndex]
            const updatedMemory: MemoryInZone = {
              ...memory,
              content: options.content ?? memory.content,
              metadata: options.metadata ?? memory.metadata,
              tags: options.tags ?? memory.tags,
              updatedAt: new Date().toISOString()
            }
            if (options.metadata) {
              updatedMemory.ui_x = typeof options.metadata.ui_x === 'number' ? options.metadata.ui_x : memory.ui_x
              updatedMemory.ui_y = typeof options.metadata.ui_y === 'number' ? options.metadata.ui_y : memory.ui_y
            }
            mockState.memories[zoneId][memoryIndex] = updatedMemory
            return updatedMemory
          }
        }
        throw new Error(`Memory not found: ${memoryId}`)
      },

      delete: async (memoryId: string): Promise<void> => {
        for (const zoneId in mockState.memories) {
          const memoryIndex = mockState.memories[zoneId].findIndex(m => m.id === memoryId)
          if (memoryIndex !== -1) {
            mockState.memories[zoneId].splice(memoryIndex, 1)
            return
          }
        }
      },

      linkTo: async (_memoryId: string, _targetId: string): Promise<void> => {
        // Mock implementation - just log for now
        console.log('Mock: linking memory', _memoryId, 'to', _targetId)
      },

      unlinkFrom: async (_memoryId: string, _targetId: string): Promise<void> => {
        // Mock implementation - just log for now
        console.log('Mock: unlinking memory', _memoryId, 'from', _targetId)
      }
    }
  }
}

export function useDatabase(): ElectronAPI {
  // Access the electron API exposed via preload
  const api = window.electronAPI
  
  if (!api) {
    console.warn('Electron API not available - running in browser mock mode')
    return createMockAPI()
  }
  
  return api
}
