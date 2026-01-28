import React from 'react'
import { ContextMenu, ContextMenuItem, ContextMenuSeparator } from './ContextMenu'
import { useDatabase } from '../../hooks/useDatabase'
import { useZones } from '../../hooks/useZones'

interface PlanContextMenuProps {
  x: number
  y: number
  planId: string
  onClose: () => void
}

export function PlanContextMenu({ x, y, planId, onClose }: PlanContextMenuProps) {
  const db = useDatabase()
  const { selectedZone, refreshSelectedZone } = useZones()
  
  const plan = selectedZone?.plans.find(p => p.id === planId)
  
  const handleRename = () => {
    // Trigger inline rename by dispatching a custom event
    const event = new CustomEvent('plan:rename', { detail: { planId } })
    window.dispatchEvent(event)
    onClose()
  }
  
  const handleDelete = async () => {
    if (!plan) {
      onClose()
      return
    }
    
    const confirmDelete = window.confirm(
      `Are you sure you want to delete "${plan.name}"? This will also delete all ${plan.tasks.length} tasks in this plan.`
    )
    
    if (confirmDelete) {
      try {
        await db.plans.delete(planId)
        refreshSelectedZone()
      } catch (err) {
        console.error('Failed to delete plan:', err)
      }
    }
    onClose()
  }
  
  const handleAddTask = async () => {
    try {
      await db.tasks.create({
        planId,
        content: 'New task',
        status: 'pending',
        metadata: { ui_x: 180, ui_y: 60 }
      })
      refreshSelectedZone()
    } catch (err) {
      console.error('Failed to create task:', err)
    }
    onClose()
  }
  
  return (
    <ContextMenu x={x} y={y} onClose={onClose}>
      <ContextMenuItem onClick={handleAddTask}>
        Add Task
      </ContextMenuItem>
      <ContextMenuSeparator />
      <ContextMenuItem onClick={handleRename}>
        Rename Plan
      </ContextMenuItem>
      <ContextMenuSeparator />
      <ContextMenuItem onClick={handleDelete} danger>
        Delete Plan
      </ContextMenuItem>
    </ContextMenu>
  )
}
