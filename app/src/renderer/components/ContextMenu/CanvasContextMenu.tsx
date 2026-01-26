import React from 'react'
import { ContextMenu, ContextMenuItem } from './ContextMenu'
import { useTasks } from '../../hooks'
import { useAppStore } from '../../stores/appStore'

interface CanvasContextMenuProps {
  x: number
  y: number
  onClose: () => void
}

export function CanvasContextMenu({ x, y, onClose }: CanvasContextMenuProps) {
  const { createTask } = useTasks()
  const { selectedPlan } = useAppStore()
  
  const handleAddTask = async () => {
    if (!selectedPlan) return
    
    try {
      // Calculate position in canvas coordinates
      // For now, use approximate position based on click location
      const metadata = {
        ui_x: x - 125, // Center the task on click
        ui_y: y - 75,
        ui_width: 250,
        ui_height: 150
      }
      
      await createTask({
        content: 'New task',
        metadata
      })
      
      onClose()
    } catch (err) {
      console.error('Failed to create task:', err)
    }
  }
  
  return (
    <ContextMenu x={x} y={y} onClose={onClose}>
      <ContextMenuItem onClick={handleAddTask}>
        <span className="mr-2">+</span>
        Add New Task
      </ContextMenuItem>
    </ContextMenu>
  )
}
