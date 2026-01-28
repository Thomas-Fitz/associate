import type { ElectronAPI } from '../../preload'
import type { Plan, TaskInPlan, PlanWithTasks, ListPlansOptions, CreateTaskOptions, UpdateTaskOptions } from '../types'

declare global {
  interface Window {
    electronAPI: ElectronAPI
  }
}

// Mock data for browser testing
const mockPlans: Plan[] = [
  {
    id: 'plan-1',
    name: 'Website Redesign',
    description: 'Redesign the company website with modern UI',
    status: 'active',
    metadata: {},
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
    metadata: {},
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
    metadata: {},
    tags: ['mobile', 'launch'],
    createdAt: '2024-01-01T09:00:00Z',
    updatedAt: '2024-01-25T18:00:00Z',
    taskCount: 4
  }
]

const mockTasks: Record<string, TaskInPlan[]> = {
  'plan-1': [
    {
      id: 'task-1-1',
      content: 'Create wireframes for homepage',
      status: 'completed',
      metadata: { ui_x: 50, ui_y: 50, ui_width: 250, ui_height: 150 },
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
      metadata: { ui_x: 350, ui_y: 50, ui_width: 250, ui_height: 150 },
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
      metadata: { ui_x: 650, ui_y: 50, ui_width: 250, ui_height: 150 },
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
      metadata: { ui_x: 50, ui_y: 50, ui_width: 250, ui_height: 150 },
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
      metadata: { ui_x: 350, ui_y: 50, ui_width: 250, ui_height: 150 },
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
      metadata: { ui_x: 50, ui_y: 50, ui_width: 250, ui_height: 150 },
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
      metadata: { ui_x: 350, ui_y: 50, ui_width: 250, ui_height: 150 },
      tags: ['android'],
      createdAt: '2024-01-01T10:00:00Z',
      updatedAt: '2024-01-06T14:00:00Z',
      position: 2000,
      dependsOn: [],
      blocks: []
    },
    {
      id: 'task-3-3',
      content: 'Monitor crash reports',
      status: 'completed',
      metadata: { ui_x: 200, ui_y: 250, ui_width: 250, ui_height: 150 },
      tags: ['monitoring'],
      createdAt: '2024-01-07T08:00:00Z',
      updatedAt: '2024-01-25T18:00:00Z',
      position: 3000,
      dependsOn: ['task-3-1', 'task-3-2'],
      blocks: []
    },
    {
      id: 'task-3-4',
      content: 'Prepare marketing materials',
      status: 'completed',
      metadata: { ui_x: 500, ui_y: 250, ui_width: 250, ui_height: 150 },
      tags: ['marketing'],
      createdAt: '2024-01-02T11:00:00Z',
      updatedAt: '2024-01-20T13:00:00Z',
      position: 4000,
      dependsOn: [],
      blocks: []
    }
  ]
}

// In-memory state for mock mode
let mockState = {
  plans: [...mockPlans],
  tasks: JSON.parse(JSON.stringify(mockTasks)) as Record<string, TaskInPlan[]>
}

function createMockAPI(): ElectronAPI {
  return {
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
