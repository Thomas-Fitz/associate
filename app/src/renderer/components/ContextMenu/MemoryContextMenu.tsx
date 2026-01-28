import React from 'react'
import { ContextMenu, ContextMenuItem, ContextMenuSeparator } from './ContextMenu'
import { useDatabase } from '../../hooks/useDatabase'
import { useZones } from '../../hooks/useZones'

interface MemoryContextMenuProps {
  x: number
  y: number
  memoryId: string
  onClose: () => void
}

export function MemoryContextMenu({ x, y, memoryId, onClose }: MemoryContextMenuProps) {
  const db = useDatabase()
  const { selectedZone, refreshSelectedZone } = useZones()
  
  const memory = selectedZone?.memories.find(m => m.id === memoryId)
  
  const handleEdit = () => {
    // Trigger inline edit by dispatching a custom event
    const event = new CustomEvent('memory:edit', { detail: { memoryId } })
    window.dispatchEvent(event)
    onClose()
  }
  
  const handleDelete = async () => {
    if (!memory) {
      onClose()
      return
    }
    
    const confirmDelete = window.confirm(
      `Are you sure you want to delete this ${memory.type.toLowerCase()}?`
    )
    
    if (confirmDelete) {
      try {
        await db.memories.delete(memoryId)
        refreshSelectedZone()
      } catch (err) {
        console.error('Failed to delete memory:', err)
      }
    }
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
