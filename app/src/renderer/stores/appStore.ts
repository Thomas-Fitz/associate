import { create } from 'zustand'
import type { Plan, TaskInPlan, PlanWithTasks, PlanStatus, EdgeInfo, Zone, ZoneWithContents } from '../types'

interface AppState {
  // Zones (new)
  zones: Zone[]
  selectedZoneId: string | null
  selectedZone: ZoneWithContents | null
  zonesLoading: boolean
  zonesError: string | null
  selectedZoneLoading: boolean
  
  // Plans (kept for backward compatibility)
  plans: Plan[]
  selectedPlanId: string | null
  selectedPlan: PlanWithTasks | null
  plansLoading: boolean
  plansError: string | null
  
  // Filtering (simplified - search only for zones)
  searchQuery: string
  statusFilter: PlanStatus | 'all'
  
  // Selection (expanded for memories)
  selectedTaskIds: Set<string>
  selectedMemoryIds: Set<string>
  selectedEdgeIds: Set<string>
  
  // Context menu (expanded for zones)
  contextMenu: {
    visible: boolean
    x: number
    y: number
    canvasX?: number
    canvasY?: number
    type: 'canvas' | 'task' | 'edge' | 'zone' | 'plan' | 'memory'
    taskId?: string
    edgeId?: string
    zoneId?: string
    planId?: string
    memoryId?: string
  } | null
  
  // Dialogs
  deleteDialog: {
    visible: boolean
    taskIds: string[]
  } | null
  
  deleteEdgeDialog: {
    visible: boolean
    edges: EdgeInfo[]
  } | null

  deleteZoneDialog: {
    visible: boolean
    zoneId: string
    zoneName: string
  } | null

  deletePlanDialog: {
    visible: boolean
    planId: string
    planName: string
    taskCount: number
  } | null

  deleteMemoryDialog: {
    visible: boolean
    memoryId: string
    memoryType: string
  } | null

  // Toast notifications
  toasts: Array<{
    id: string
    type: 'success' | 'error' | 'info' | 'warning'
    message: string
    duration?: number
  }>

  // Toast Actions
  addToast: (toast: { type: 'success' | 'error' | 'info' | 'warning'; message: string; duration?: number }) => string
  removeToast: (id: string) => void
  
  // Zone Actions
  setZones: (zones: Zone[]) => void
  setSelectedZoneId: (zoneId: string | null) => void
  setSelectedZone: (zone: ZoneWithContents | null) => void
  setZonesLoading: (loading: boolean) => void
  setZonesError: (error: string | null) => void
  setSelectedZoneLoading: (loading: boolean) => void
  
  // Plan Actions
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
  
  setSelectedMemoryIds: (ids: Set<string>) => void
  toggleMemorySelection: (memoryId: string, addToSelection?: boolean) => void
  clearMemorySelection: () => void
  
  setSelectedEdgeIds: (ids: Set<string>) => void
  clearEdgeSelection: () => void
  
  showContextMenu: (x: number, y: number, type: 'canvas' | 'task' | 'edge' | 'zone' | 'plan' | 'memory', options?: { taskId?: string; canvasX?: number; canvasY?: number; edgeId?: string; zoneId?: string; planId?: string; memoryId?: string }) => void
  hideContextMenu: () => void
  
  showDeleteDialog: (taskIds: string[]) => void
  hideDeleteDialog: () => void
  
  showDeleteEdgeDialog: (edges: EdgeInfo[]) => void
  hideDeleteEdgeDialog: () => void

  showDeleteZoneDialog: (zoneId: string, zoneName: string) => void
  hideDeleteZoneDialog: () => void

  showDeletePlanDialog: (planId: string, planName: string, taskCount: number) => void
  hideDeletePlanDialog: () => void

  showDeleteMemoryDialog: (memoryId: string, memoryType: string) => void
  hideDeleteMemoryDialog: () => void
  
  // Task updates
  updateTask: (taskId: string, updates: Partial<TaskInPlan>) => void
  addTask: (task: TaskInPlan) => void
  removeTask: (taskId: string) => void
  removeTasks: (taskIds: string[]) => void
}

export const useAppStore = create<AppState>((set, get) => ({
  // Initial state
  zones: [],
  selectedZoneId: null,
  selectedZone: null,
  zonesLoading: false,
  zonesError: null,
  selectedZoneLoading: false,
  
  plans: [],
  selectedPlanId: null,
  selectedPlan: null,
  plansLoading: false,
  plansError: null,
  
  searchQuery: '',
  statusFilter: 'all',
  
  selectedTaskIds: new Set(),
  selectedMemoryIds: new Set(),
  selectedEdgeIds: new Set(),
  
  contextMenu: null,
  deleteDialog: null,
  deleteEdgeDialog: null,
  deleteZoneDialog: null,
  deletePlanDialog: null,
  deleteMemoryDialog: null,
  toasts: [],
  
  // Zone Actions
  setZones: (zones) => set({ zones }),
  setSelectedZoneId: (zoneId) => set({ selectedZoneId: zoneId }),
  setSelectedZone: (zone) => set({ selectedZone: zone }),
  setZonesLoading: (loading) => set({ zonesLoading: loading }),
  setZonesError: (error) => set({ zonesError: error }),
  setSelectedZoneLoading: (loading) => set({ selectedZoneLoading: loading }),
  
  // Plan Actions
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
  
  setSelectedMemoryIds: (ids) => set({ selectedMemoryIds: ids }),
  
  toggleMemorySelection: (memoryId, addToSelection = false) => {
    const { selectedMemoryIds } = get()
    const newSelection = new Set(addToSelection ? selectedMemoryIds : [])
    
    if (newSelection.has(memoryId)) {
      newSelection.delete(memoryId)
    } else {
      newSelection.add(memoryId)
    }
    
    set({ selectedMemoryIds: newSelection })
  },
  
  clearMemorySelection: () => set({ selectedMemoryIds: new Set() }),
  
  setSelectedEdgeIds: (ids) => set({ selectedEdgeIds: ids }),
  
  clearEdgeSelection: () => set({ selectedEdgeIds: new Set() }),
  
  showContextMenu: (x, y, type, options = {}) => set({
    contextMenu: { 
      visible: true, 
      x, 
      y, 
      type, 
      taskId: options.taskId, 
      canvasX: options.canvasX, 
      canvasY: options.canvasY, 
      edgeId: options.edgeId,
      zoneId: options.zoneId,
      planId: options.planId,
      memoryId: options.memoryId
    }
  }),
  
  hideContextMenu: () => set({ contextMenu: null }),
  
  showDeleteDialog: (taskIds) => set({
    deleteDialog: { visible: true, taskIds }
  }),
  
  hideDeleteDialog: () => set({ deleteDialog: null }),
  
  showDeleteEdgeDialog: (edges) => set({
    deleteEdgeDialog: { visible: true, edges }
  }),
  
  hideDeleteEdgeDialog: () => set({ deleteEdgeDialog: null }),

  showDeleteZoneDialog: (zoneId, zoneName) => set({
    deleteZoneDialog: { visible: true, zoneId, zoneName }
  }),

  hideDeleteZoneDialog: () => set({ deleteZoneDialog: null }),

  showDeletePlanDialog: (planId, planName, taskCount) => set({
    deletePlanDialog: { visible: true, planId, planName, taskCount }
  }),

  hideDeletePlanDialog: () => set({ deletePlanDialog: null }),

  showDeleteMemoryDialog: (memoryId, memoryType) => set({
    deleteMemoryDialog: { visible: true, memoryId, memoryType }
  }),

  hideDeleteMemoryDialog: () => set({ deleteMemoryDialog: null }),
  
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
  },

  // Toast Actions
  addToast: (toast) => {
    const id = crypto.randomUUID()
    set((state) => ({
      toasts: [...state.toasts, { ...toast, id }]
    }))
    
    // Auto-remove after duration (default 5 seconds)
    const duration = toast.duration ?? 5000
    if (duration > 0) {
      setTimeout(() => {
        get().removeToast(id)
      }, duration)
    }
    
    return id
  },
  
  removeToast: (id) => set((state) => ({
    toasts: state.toasts.filter(t => t.id !== id)
  }))
}))
