import { useCallback } from 'react'
import { useAppStore } from '../stores/appStore'
import { useDatabase } from './useDatabase'
import type { MemoryInZone, PlanInZone, TaskInPlan } from '../types'

/**
 * Node types that can be created on the zone canvas.
 * Excludes Zone since zones are the container - you can't add a zone to a zone.
 */
export type CanvasNodeType = 'plan' | 'task' | 'memory'

/**
 * Memory subtypes that can be created
 */
export type MemoryType = 'Note' | 'Repository' | 'Memory'

interface NodeCreationPosition {
  x: number
  y: number
}

interface CreateNodeOptions {
  position: NodeCreationPosition
}

interface CreatePlanResult {
  type: 'plan'
  node: PlanInZone
}

interface CreateTaskResult {
  type: 'task'
  node: TaskInPlan
}

interface CreateMemoryResult {
  type: 'memory'
  node: MemoryInZone
}

export type CreateNodeResult = CreatePlanResult | CreateTaskResult | CreateMemoryResult

/**
 * Hook for creating nodes on the zone canvas.
 * Provides a unified interface for creating plans, tasks, and memories
 * at a specific position on the canvas.
 * 
 * This hook follows the single responsibility principle - it only handles
 * node creation on the canvas, delegating actual DB operations to the
 * appropriate API methods.
 */
export function useCanvasNodeCreation() {
  const db = useDatabase()
  const { selectedZone, selectedPlan, setSelectedZone, addTask } = useAppStore()

  /**
   * Create a plan at the specified canvas position
   */
  const createPlan = useCallback(async (options: CreateNodeOptions): Promise<PlanInZone | null> => {
    if (!selectedZone) {
      console.error('No zone selected')
      return null
    }

    const metadata = {
      ui_x: options.position.x,
      ui_y: options.position.y,
      ui_width: 400,
      ui_height: 300
    }

    try {
      const plan = await db.plans.create({
        zoneId: selectedZone.id,
        name: 'New Plan',
        description: '',
        status: 'draft',
        metadata
      })

      // Create the PlanInZone representation
      const planInZone: PlanInZone = {
        id: plan.id,
        name: plan.name,
        description: plan.description,
        status: plan.status,
        metadata: plan.metadata as PlanInZone['metadata'],
        tags: plan.tags,
        createdAt: plan.createdAt,
        updatedAt: plan.updatedAt,
        tasks: []
      }

      // Update local state
      setSelectedZone({
        ...selectedZone,
        plans: [...selectedZone.plans, planInZone],
        planCount: (selectedZone.planCount || 0) + 1
      })

      return planInZone
    } catch (err) {
      console.error('Failed to create plan:', err)
      throw err
    }
  }, [db.plans, selectedZone, setSelectedZone])

  /**
   * Create a task at the specified canvas position.
   * Requires a plan to be selected.
   */
  const createTask = useCallback(async (options: CreateNodeOptions): Promise<TaskInPlan | null> => {
    if (!selectedPlan) {
      console.error('No plan selected')
      return null
    }

    const metadata = {
      ui_x: options.position.x,
      ui_y: options.position.y,
      ui_width: 250,
      ui_height: 150
    }

    try {
      const task = await db.tasks.create({
        planId: selectedPlan.id,
        content: 'New task',
        status: 'pending',
        metadata
      })

      // Add to local state
      addTask(task)

      return task
    } catch (err) {
      console.error('Failed to create task:', err)
      throw err
    }
  }, [db.tasks, selectedPlan, addTask])

  /**
   * Create a memory at the specified canvas position
   */
  const createMemory = useCallback(async (
    options: CreateNodeOptions,
    memoryType: MemoryType = 'Note'
  ): Promise<MemoryInZone | null> => {
    if (!selectedZone) {
      console.error('No zone selected')
      return null
    }

    const metadata = {
      ui_x: options.position.x,
      ui_y: options.position.y
    }

    try {
      const memory = await db.memories.create({
        zoneId: selectedZone.id,
        type: memoryType,
        content: `New ${memoryType.toLowerCase()}`,
        metadata
      })

      // Update local state
      setSelectedZone({
        ...selectedZone,
        memories: [...selectedZone.memories, memory],
        memoryCount: (selectedZone.memoryCount || 0) + 1
      })

      return memory
    } catch (err) {
      console.error('Failed to create memory:', err)
      throw err
    }
  }, [db.memories, selectedZone, setSelectedZone])

  /**
   * Check if a node type can be created in the current context
   */
  const canCreateNodeType = useCallback((nodeType: CanvasNodeType): boolean => {
    switch (nodeType) {
      case 'plan':
        return !!selectedZone
      case 'task':
        return !!selectedPlan
      case 'memory':
        return !!selectedZone
      default:
        return false
    }
  }, [selectedZone, selectedPlan])

  /**
   * Get a human-readable reason why a node type cannot be created
   */
  const getCannotCreateReason = useCallback((nodeType: CanvasNodeType): string | null => {
    switch (nodeType) {
      case 'plan':
        return selectedZone ? null : 'No zone selected'
      case 'task':
        return selectedPlan ? null : 'No plan selected'
      case 'memory':
        return selectedZone ? null : 'No zone selected'
      default:
        return 'Unknown node type'
    }
  }, [selectedZone, selectedPlan])

  return {
    createPlan,
    createTask,
    createMemory,
    canCreateNodeType,
    getCannotCreateReason,
    hasSelectedZone: !!selectedZone,
    hasSelectedPlan: !!selectedPlan
  }
}
