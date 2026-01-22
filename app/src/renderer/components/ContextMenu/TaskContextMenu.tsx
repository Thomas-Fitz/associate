import React from 'react'
import { ContextMenu, ContextMenuItem, ContextMenuSeparator, ContextMenuSubMenu } from './ContextMenu'
import { useTasks, useSelection } from '../../hooks'
import type { TaskStatus } from '../../types'

interface TaskContextMenuProps {
  x: number
  y: number
  taskId: string
  onClose: () => void
}

const statusOptions: { value: TaskStatus; label: string }[] = [
  { value: 'pending', label: 'Pending' },
  { value: 'in_progress', label: 'In Progress' },
  { value: 'completed', label: 'Completed' },
  { value: 'cancelled', label: 'Cancelled' },
  { value: 'blocked', label: 'Blocked' }
]

export function TaskContextMenu({ x, y, taskId, onClose }: TaskContextMenuProps) {
  const { updateTaskStatus, updateTasksStatus, confirmDelete } = useTasks()
  const { selectedTaskIds, isSelected, selectTask } = useSelection()
  
  // Check if this task is part of a multi-selection
  const isMultiSelect = selectedTaskIds.size > 1 && isSelected(taskId)
  const targetTaskIds = isMultiSelect ? Array.from(selectedTaskIds) : [taskId]
  
  const handleStatusChange = async (status: TaskStatus) => {
    try {
      if (isMultiSelect) {
        await updateTasksStatus(targetTaskIds, status)
      } else {
        await updateTaskStatus(taskId, status)
      }
      onClose()
    } catch (err) {
      console.error('Failed to update status:', err)
    }
  }
  
  const handleDelete = () => {
    // If task is not selected but we're right-clicking on it, select it first
    if (!isSelected(taskId)) {
      selectTask(taskId)
    }
    confirmDelete(targetTaskIds)
    onClose()
  }
  
  return (
    <ContextMenu x={x} y={y} onClose={onClose}>
      <ContextMenuSubMenu label="Set Status">
        {statusOptions.map(({ value, label }) => (
          <ContextMenuItem key={value} onClick={() => handleStatusChange(value)}>
            {label}
          </ContextMenuItem>
        ))}
      </ContextMenuSubMenu>
      
      <ContextMenuSeparator />
      
      <ContextMenuItem onClick={handleDelete} danger>
        {isMultiSelect 
          ? `Delete Selected (${selectedTaskIds.size})` 
          : 'Delete Task'
        }
      </ContextMenuItem>
    </ContextMenu>
  )
}
