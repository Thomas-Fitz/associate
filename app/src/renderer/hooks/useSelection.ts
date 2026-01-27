import { useCallback } from 'react'
import { useAppStore } from '../stores/appStore'

export function useSelection() {
  const {
    selectedTaskIds,
    setSelectedTaskIds,
    toggleTaskSelection,
    clearTaskSelection
  } = useAppStore()
  
  // Select a single task (clear others)
  const selectTask = useCallback((taskId: string) => {
    setSelectedTaskIds(new Set([taskId]))
  }, [setSelectedTaskIds])
  
  // Select multiple tasks (replace selection)
  const selectTasks = useCallback((taskIds: string[]) => {
    setSelectedTaskIds(new Set(taskIds))
  }, [setSelectedTaskIds])
  
  // Add task to selection (Ctrl+click behavior)
  const addToSelection = useCallback((taskId: string) => {
    toggleTaskSelection(taskId, true)
  }, [toggleTaskSelection])
  
  // Check if a task is selected
  const isSelected = useCallback((taskId: string) => {
    return selectedTaskIds.has(taskId)
  }, [selectedTaskIds])
  
  // Get selected task IDs as array
  const getSelectedTaskIds = useCallback(() => {
    return Array.from(selectedTaskIds)
  }, [selectedTaskIds])
  
  return {
    selectedTaskIds,
    selectedCount: selectedTaskIds.size,
    selectTask,
    selectTasks,
    addToSelection,
    toggleTaskSelection,
    clearSelection: clearTaskSelection,
    isSelected,
    getSelectedTaskIds
  }
}
