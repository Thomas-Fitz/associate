import React from 'react'
import { ContextMenu, ContextMenuItem } from './ContextMenu'
import { useTasks } from '../../hooks'
import { useAppStore } from '../../stores/appStore'

interface CanvasContextMenuProps {
  x: number
  y: number
  canvasX?: number
  canvasY?: number
  onClose: () => void
}

export function CanvasContextMenu({ x, y, canvasX, canvasY, onClose }: CanvasContextMenuProps) {
  const { createTask } = useTasks()
  const { selectedPlan } = useAppStore()
  
  const handleAddTask = async () => {
    if (!selectedPlan) return
    
    try {
      // Use canvas coordinates if available, otherwise fall back to screen coords
      const taskX = canvasX ?? x - 125
      const taskY = canvasY ?? y - 75
      
      const metadata = {
        ui_x: taskX - 125, // Center the task on the click position (task is 250px wide)
        ui_y: taskY - 75,  // Center vertically (task is ~150px tall)
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
