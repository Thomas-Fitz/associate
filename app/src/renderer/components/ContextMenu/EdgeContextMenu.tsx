import React from 'react'
import { ContextMenu, ContextMenuItem } from './ContextMenu'
import { useEdges } from '../../hooks'

interface EdgeContextMenuProps {
  x: number
  y: number
  edgeId: string
  onClose: () => void
}

export function EdgeContextMenu({ x, y, edgeId, onClose }: EdgeContextMenuProps) {
  const { selectedEdgeIds, isEdgeSelected, selectEdge, confirmDeleteEdges } = useEdges()
  
  // Check if this edge is part of a multi-selection
  const isMultiSelect = selectedEdgeIds.size > 1 && isEdgeSelected(edgeId)
  const targetEdgeIds = isMultiSelect ? Array.from(selectedEdgeIds) : [edgeId]
  
  const handleDelete = () => {
    // If edge is not selected but we're right-clicking on it, select it first
    if (!isEdgeSelected(edgeId)) {
      selectEdge(edgeId)
    }
    confirmDeleteEdges(targetEdgeIds)
    onClose()
  }
  
  return (
    <ContextMenu x={x} y={y} onClose={onClose}>
      <ContextMenuItem onClick={handleDelete} danger>
        {isMultiSelect 
          ? `Delete Selected (${selectedEdgeIds.size})` 
          : 'Delete Relationship'
        }
      </ContextMenuItem>
    </ContextMenu>
  )
}
