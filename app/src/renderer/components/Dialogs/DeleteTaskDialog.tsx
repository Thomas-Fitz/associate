import React from 'react'
import { ConfirmDialog } from './ConfirmDialog'
import { useTasks } from '../../hooks'
import { useAppStore } from '../../stores/appStore'

export function DeleteTaskDialog() {
  const { deleteDialog, hideDeleteDialog, selectedPlan } = useAppStore()
  const { deleteTasks } = useTasks()
  
  if (!deleteDialog?.visible || !deleteDialog.taskIds.length) {
    return null
  }
  
  const taskCount = deleteDialog.taskIds.length
  const isMultiple = taskCount > 1
  
  // Get task content for single delete
  let taskDescription = ''
  if (!isMultiple && selectedPlan) {
    const task = selectedPlan.tasks.find(t => t.id === deleteDialog.taskIds[0])
    if (task) {
      taskDescription = task.content.length > 50 
        ? task.content.substring(0, 50) + '...' 
        : task.content
    }
  }
  
  const handleConfirm = async () => {
    await deleteTasks(deleteDialog.taskIds)
  }
  
  return (
    <ConfirmDialog
      title={isMultiple ? `Delete ${taskCount} Tasks` : 'Delete Task'}
      message={
        isMultiple
          ? `Are you sure you want to delete ${taskCount} tasks? This action is permanent and cannot be undone.`
          : `Are you sure you want to delete this task? This action is permanent and cannot be undone.${
              taskDescription ? `\n\nTask: "${taskDescription}"` : ''
            }`
      }
      confirmLabel={isMultiple ? 'Delete All' : 'Delete'}
      danger
      onConfirm={handleConfirm}
      onCancel={hideDeleteDialog}
    />
  )
}
