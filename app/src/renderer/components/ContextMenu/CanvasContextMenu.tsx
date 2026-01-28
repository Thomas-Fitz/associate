import React from 'react'
import { ContextMenu, ContextMenuItem, ContextMenuSeparator, ContextMenuSubMenu } from './ContextMenu'
import { useCanvasNodeCreation, type MemoryType } from '../../hooks/useCanvasNodeCreation'

interface CanvasContextMenuProps {
  x: number
  y: number
  canvasX?: number
  canvasY?: number
  onClose: () => void
}

/**
 * Context menu shown when right-clicking on the zone canvas.
 * Provides options to create different node types (Plan, Task, Memory).
 * Zone is excluded since you cannot add a zone inside a zone.
 */
export function CanvasContextMenu({ x, y, canvasX, canvasY, onClose }: CanvasContextMenuProps) {
  const {
    createPlan,
    createTask,
    createMemory,
    canCreateNodeType,
    getCannotCreateReason
  } = useCanvasNodeCreation()

  // Calculate position - center the node on the click position
  const getNodePosition = (nodeWidth: number, nodeHeight: number) => {
    const posX = canvasX ?? x
    const posY = canvasY ?? y
    return {
      x: posX - nodeWidth / 2,
      y: posY - nodeHeight / 2
    }
  }

  const handleAddPlan = async () => {
    try {
      // Plan default size: 400x300
      const position = getNodePosition(400, 300)
      await createPlan({ position })
      onClose()
    } catch (err) {
      console.error('Failed to create plan:', err)
    }
  }

  const handleAddTask = async () => {
    try {
      // Task default size: 250x150
      const position = getNodePosition(250, 150)
      await createTask({ position })
      onClose()
    } catch (err) {
      console.error('Failed to create task:', err)
    }
  }

  const handleAddMemory = async (memoryType: MemoryType) => {
    try {
      // Memory nodes are smaller, default to 200x100
      const position = getNodePosition(200, 100)
      await createMemory({ position }, memoryType)
      onClose()
    } catch (err) {
      console.error('Failed to create memory:', err)
    }
  }

  const canCreatePlan = canCreateNodeType('plan')
  const canCreateTaskNode = canCreateNodeType('task')
  const canCreateMemoryNode = canCreateNodeType('memory')

  const planDisabledReason = getCannotCreateReason('plan')
  const taskDisabledReason = getCannotCreateReason('task')
  const memoryDisabledReason = getCannotCreateReason('memory')

  return (
    <ContextMenu x={x} y={y} onClose={onClose}>
      <ContextMenuItem 
        onClick={handleAddPlan}
        disabled={!canCreatePlan}
      >
        <span className="mr-2">+</span>
        Add New Plan
        {!canCreatePlan && planDisabledReason && (
          <span className="ml-2 text-xs text-surface-400">({planDisabledReason})</span>
        )}
      </ContextMenuItem>
      
      <ContextMenuItem 
        onClick={handleAddTask}
        disabled={!canCreateTaskNode}
      >
        <span className="mr-2">+</span>
        Add New Task
        {!canCreateTaskNode && taskDisabledReason && (
          <span className="ml-2 text-xs text-surface-400">({taskDisabledReason})</span>
        )}
      </ContextMenuItem>
      
      <ContextMenuSeparator />
      
      <ContextMenuSubMenu label="Add Memory">
        <ContextMenuItem 
          onClick={() => handleAddMemory('Note')}
          disabled={!canCreateMemoryNode}
        >
          <span className="mr-2">📝</span>
          Note
          {!canCreateMemoryNode && memoryDisabledReason && (
            <span className="ml-2 text-xs text-surface-400">({memoryDisabledReason})</span>
          )}
        </ContextMenuItem>
        <ContextMenuItem 
          onClick={() => handleAddMemory('Repository')}
          disabled={!canCreateMemoryNode}
        >
          <span className="mr-2">📁</span>
          Repository
        </ContextMenuItem>
        <ContextMenuItem 
          onClick={() => handleAddMemory('Memory')}
          disabled={!canCreateMemoryNode}
        >
          <span className="mr-2">💭</span>
          Memory
        </ContextMenuItem>
      </ContextMenuSubMenu>
    </ContextMenu>
  )
}
