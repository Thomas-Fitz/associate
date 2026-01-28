import React from 'react'
import { ContextMenu, ContextMenuItem, ContextMenuSeparator, ContextMenuSubMenu } from './ContextMenu'
import { useDatabase } from '../../hooks/useDatabase'
import { useZones } from '../../hooks/useZones'
import { useToast } from '../../hooks'
import { useAppStore } from '../../stores/appStore'

interface PlanContextMenuProps {
  x: number
  y: number
  planId: string
  onClose: () => void
}

export function PlanContextMenu({ x, y, planId, onClose }: PlanContextMenuProps) {
  const db = useDatabase()
  const { zones, selectedZone, selectedZoneId, refreshSelectedZone, refreshZones } = useZones()
  const { showDeletePlanDialog } = useAppStore()
  const toast = useToast()
  
  const plan = selectedZone?.plans.find(p => p.id === planId)
  
  // Get other zones to move to (exclude current zone)
  const otherZones = zones.filter(z => z.id !== selectedZoneId)
  
  const handleRename = () => {
    // Trigger inline rename by dispatching a custom event
    const event = new CustomEvent('plan:rename', { detail: { planId } })
    window.dispatchEvent(event)
    onClose()
  }
  
  const handleDelete = () => {
    if (!plan) {
      onClose()
      return
    }
    
    showDeletePlanDialog(planId, plan.name, plan.tasks.length)
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
  
  const handleMoveToZone = async (targetZoneId: string) => {
    try {
      await db.plans.move(planId, targetZoneId)
      // Refresh both zones list and selected zone
      await refreshZones()
      refreshSelectedZone()
      toast.success('Plan moved successfully')
    } catch (err) {
      console.error('Failed to move plan:', err)
      toast.error('Failed to move plan')
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
      {otherZones.length > 0 && (
        <ContextMenuSubMenu label="Move to Zone">
          {otherZones.map(zone => (
            <ContextMenuItem key={zone.id} onClick={() => handleMoveToZone(zone.id)}>
              {zone.name}
            </ContextMenuItem>
          ))}
        </ContextMenuSubMenu>
      )}
      <ContextMenuSeparator />
      <ContextMenuItem onClick={handleDelete} danger>
        Delete Plan
      </ContextMenuItem>
    </ContextMenu>
  )
}
