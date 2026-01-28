import React from 'react'
import { ContextMenu, ContextMenuItem, ContextMenuSeparator } from './ContextMenu'
import { useZones } from '../../hooks/useZones'
import { useAppStore } from '../../stores/appStore'

interface MemoryContextMenuProps {
  x: number
  y: number
  memoryId: string
  onClose: () => void
}

export function MemoryContextMenu({ x, y, memoryId, onClose }: MemoryContextMenuProps) {
  const { selectedZone } = useZones()
  const { showDeleteMemoryDialog } = useAppStore()
  
  const memory = selectedZone?.memories.find(m => m.id === memoryId)
  
  const handleEdit = () => {
    // Trigger inline edit by dispatching a custom event
    const event = new CustomEvent('memory:edit', { detail: { memoryId } })
    window.dispatchEvent(event)
    onClose()
  }
  
  const handleDelete = () => {
    if (!memory) {
      onClose()
      return
    }
    
    showDeleteMemoryDialog(memoryId, memory.type)
    onClose()
  }
  
  return (
    <ContextMenu x={x} y={y} onClose={onClose}>
      <ContextMenuItem onClick={handleEdit}>
        Edit {memory?.type || 'Memory'}
      </ContextMenuItem>
      <ContextMenuSeparator />
      <ContextMenuItem onClick={handleDelete} danger>
        Delete {memory?.type || 'Memory'}
      </ContextMenuItem>
    </ContextMenu>
  )
}
