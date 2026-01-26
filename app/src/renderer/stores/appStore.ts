import { create } from 'zustand'
import type { Plan, TaskInPlan, PlanWithTasks, PlanStatus } from '../types'

interface AppState {
  // Plans
  plans: Plan[]
  selectedPlanId: string | null
  selectedPlan: PlanWithTasks | null
  plansLoading: boolean
  plansError: string | null
  
  // Filtering
  searchQuery: string
  statusFilter: PlanStatus | 'all'
  
  // Selection
  selectedTaskIds: Set<string>
  
  // Context menu
  contextMenu: {
    visible: boolean
    x: number
    y: number
    type: 'canvas' | 'task'
    taskId?: string
  } | null
  
  // Dialogs
  deleteDialog: {
    visible: boolean
    taskIds: string[]
  } | null
  
  // Actions
  setPlans: (plans: Plan[]) => void
  setSelectedPlanId: (planId: string | null) => void
  setSelectedPlan: (plan: PlanWithTasks | null) => void
  setPlansLoading: (loading: boolean) => void
  setPlansError: (error: string | null) => void
  
  setSearchQuery: (query: string) => void
  setStatusFilter: (status: PlanStatus | 'all') => void
  
  setSelectedTaskIds: (ids: Set<string>) => void
  toggleTaskSelection: (taskId: string, addToSelection?: boolean) => void
  clearTaskSelection: () => void
  
  showContextMenu: (x: number, y: number, type: 'canvas' | 'task', taskId?: string) => void
  hideContextMenu: () => void
  
  showDeleteDialog: (taskIds: string[]) => void
  hideDeleteDialog: () => void
  
  // Task updates
  updateTask: (taskId: string, updates: Partial<TaskInPlan>) => void
  addTask: (task: TaskInPlan) => void
  removeTask: (taskId: string) => void
  removeTasks: (taskIds: string[]) => void
}

export const useAppStore = create<AppState>((set, get) => ({
  // Initial state
  plans: [],
  selectedPlanId: null,
  selectedPlan: null,
  plansLoading: false,
  plansError: null,
  
  searchQuery: '',
  statusFilter: 'all',
  
  selectedTaskIds: new Set(),
  
  contextMenu: null,
  deleteDialog: null,
  
  // Actions
  setPlans: (plans) => set({ plans }),
  setSelectedPlanId: (planId) => set({ selectedPlanId: planId }),
  setSelectedPlan: (plan) => set({ selectedPlan: plan }),
  setPlansLoading: (loading) => set({ plansLoading: loading }),
  setPlansError: (error) => set({ plansError: error }),
  
  setSearchQuery: (query) => set({ searchQuery: query }),
  setStatusFilter: (status) => set({ statusFilter: status }),
  
  setSelectedTaskIds: (ids) => set({ selectedTaskIds: ids }),
  
  toggleTaskSelection: (taskId, addToSelection = false) => {
    const { selectedTaskIds } = get()
    const newSelection = new Set(addToSelection ? selectedTaskIds : [])
    
    if (newSelection.has(taskId)) {
      newSelection.delete(taskId)
    } else {
      newSelection.add(taskId)
    }
    
    set({ selectedTaskIds: newSelection })
  },
  
  clearTaskSelection: () => set({ selectedTaskIds: new Set() }),
  
  showContextMenu: (x, y, type, taskId) => set({
    contextMenu: { visible: true, x, y, type, taskId }
  }),
  
  hideContextMenu: () => set({ contextMenu: null }),
  
  showDeleteDialog: (taskIds) => set({
    deleteDialog: { visible: true, taskIds }
  }),
  
  hideDeleteDialog: () => set({ deleteDialog: null }),
  
  updateTask: (taskId, updates) => {
    const { selectedPlan } = get()
    if (!selectedPlan) return
    
    const updatedTasks = selectedPlan.tasks.map(task =>
      task.id === taskId ? { ...task, ...updates } : task
    )
    
    set({
      selectedPlan: { ...selectedPlan, tasks: updatedTasks }
    })
  },
  
  addTask: (task) => {
    const { selectedPlan } = get()
    if (!selectedPlan) return
    
    set({
      selectedPlan: {
        ...selectedPlan,
        tasks: [...selectedPlan.tasks, task]
      }
    })
  },
  
  removeTask: (taskId) => {
    const { selectedPlan, selectedTaskIds } = get()
    if (!selectedPlan) return
    
    const newSelection = new Set(selectedTaskIds)
    newSelection.delete(taskId)
    
    set({
      selectedPlan: {
        ...selectedPlan,
        tasks: selectedPlan.tasks.filter(t => t.id !== taskId)
      },
      selectedTaskIds: newSelection
    })
  },
  
  removeTasks: (taskIds) => {
    const { selectedPlan, selectedTaskIds } = get()
    if (!selectedPlan) return
    
    const idsSet = new Set(taskIds)
    const newSelection = new Set(selectedTaskIds)
    taskIds.forEach(id => newSelection.delete(id))
    
    set({
      selectedPlan: {
        ...selectedPlan,
        tasks: selectedPlan.tasks.filter(t => !idsSet.has(t.id))
      },
      selectedTaskIds: newSelection
    })
  }
}))
