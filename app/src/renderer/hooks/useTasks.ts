import { useCallback } from 'react'
import { useAppStore } from '../stores/appStore'
import { useDatabase } from './useDatabase'
import type { TaskStatus, CreateTaskOptions, UpdateTaskOptions } from '../types'

export function useTasks() {
  const db = useDatabase()
  const {
    selectedPlan,
    selectedTaskIds,
    updateTask,
    addTask,
    removeTask,
    removeTasks,
    setSelectedTaskIds,
    showDeleteDialog,
    hideDeleteDialog
  } = useAppStore()
  
  // Create a new task
  const createTask = useCallback(async (options: Omit<CreateTaskOptions, 'planId'>) => {
    if (!selectedPlan) {
      throw new Error('No plan selected')
    }
    
    const task = await db.tasks.create({
      ...options,
      planId: selectedPlan.id
    })
    
    addTask(task)
    return task
  }, [db.tasks, selectedPlan, addTask])
  
  // Update a task
  const updateTaskContent = useCallback(async (taskId: string, options: UpdateTaskOptions) => {
    const updatedTask = await db.tasks.update(taskId, options)
    updateTask(taskId, updatedTask)
    return updatedTask
  }, [db.tasks, updateTask])
  
  // Update task position (UI metadata)
  const updateTaskPosition = useCallback(async (taskId: string, x: number, y: number) => {
    const task = selectedPlan?.tasks.find(t => t.id === taskId)
    if (!task) return
    
    const newMetadata = {
      ...task.metadata,
      ui_x: x,
      ui_y: y
    }
    
    // Update locally first for responsiveness
    updateTask(taskId, { metadata: newMetadata })
    
    // Then persist to database
    await db.tasks.update(taskId, { metadata: newMetadata })
  }, [db.tasks, selectedPlan, updateTask])
  
  // Update task size (UI metadata)
  const updateTaskSize = useCallback(async (taskId: string, width: number, height: number) => {
    const task = selectedPlan?.tasks.find(t => t.id === taskId)
    if (!task) return
    
    const newMetadata = {
      ...task.metadata,
      ui_width: width,
      ui_height: height
    }
    
    // Update locally first for responsiveness
    updateTask(taskId, { metadata: newMetadata })
    
    // Then persist to database
    await db.tasks.update(taskId, { metadata: newMetadata })
  }, [db.tasks, selectedPlan, updateTask])
  
  // Update task status
  const updateTaskStatus = useCallback(async (taskId: string, status: TaskStatus) => {
    const updatedTask = await db.tasks.update(taskId, { status })
    updateTask(taskId, { status: updatedTask.status })
    return updatedTask
  }, [db.tasks, updateTask])
  
  // Update multiple tasks' status
  const updateTasksStatus = useCallback(async (taskIds: string[], status: TaskStatus) => {
    await Promise.all(
      taskIds.map(taskId => updateTaskStatus(taskId, status))
    )
  }, [updateTaskStatus])
  
  // Delete a single task
  const deleteTask = useCallback(async (taskId: string) => {
    await db.tasks.delete(taskId)
    removeTask(taskId)
  }, [db.tasks, removeTask])
  
  // Delete multiple tasks
  const deleteTasks = useCallback(async (taskIds: string[]) => {
    await Promise.all(taskIds.map(id => db.tasks.delete(id)))
    removeTasks(taskIds)
    hideDeleteDialog()
  }, [db.tasks, removeTasks, hideDeleteDialog])
  
  // Confirm delete (shows dialog)
  const confirmDelete = useCallback((taskIds?: string[]) => {
    const ids = taskIds || Array.from(selectedTaskIds)
    if (ids.length === 0) return
    showDeleteDialog(ids)
  }, [selectedTaskIds, showDeleteDialog])
  
  // Create dependency between tasks
  const createDependency = useCallback(async (sourceTaskId: string, targetTaskId: string) => {
    try {
      await db.dependencies.create(sourceTaskId, targetTaskId)
      
      // Update local state to reflect new dependency
      const sourceTask = selectedPlan?.tasks.find(t => t.id === sourceTaskId)
      if (sourceTask) {
        updateTask(sourceTaskId, {
          dependsOn: [...sourceTask.dependsOn, targetTaskId]
        })
      }
    } catch (err) {
      if (err instanceof Error && err.message.includes('circular')) {
        throw new Error('Cannot create circular dependency')
      }
      throw err
    }
  }, [db.dependencies, selectedPlan, updateTask])
  
  // Delete dependency
  const deleteDependency = useCallback(async (sourceTaskId: string, targetTaskId: string) => {
    await db.dependencies.delete(sourceTaskId, targetTaskId)
    
    // Update local state
    const sourceTask = selectedPlan?.tasks.find(t => t.id === sourceTaskId)
    if (sourceTask) {
      updateTask(sourceTaskId, {
        dependsOn: sourceTask.dependsOn.filter(id => id !== targetTaskId)
      })
    }
  }, [db.dependencies, selectedPlan, updateTask])
  
  return {
    tasks: selectedPlan?.tasks || [],
    selectedTaskIds,
    setSelectedTaskIds,
    createTask,
    updateTaskContent,
    updateTaskPosition,
    updateTaskSize,
    updateTaskStatus,
    updateTasksStatus,
    deleteTask,
    deleteTasks,
    confirmDelete,
    createDependency,
    deleteDependency
  }
}
