import React from 'react'
import { ContextMenu, ContextMenuItem, ContextMenuSeparator } from './ContextMenu'
import { useZones } from '../../hooks'
import { useAppStore } from '../../stores/appStore'

interface ZoneContextMenuProps {
  x: number
  y: number
  zoneId: string
  onClose: () => void
}

export function ZoneContextMenu({ x, y, zoneId, onClose }: ZoneContextMenuProps) {
  const { zones, deleteZone } = useZones()
  const { showDeleteZoneDialog } = useAppStore()
  
  const zone = zones.find(z => z.id === zoneId)
  
  const handleRename = () => {
    // Trigger inline rename by dispatching a custom event
    // The ZoneList component will handle this
    const event = new CustomEvent('zone:rename', { detail: { zoneId } })
    window.dispatchEvent(event)
    onClose()
  }
  
  const handleDelete = () => {
    if (zone) {
      showDeleteZoneDialog(zoneId, zone.name)
    }
    onClose()
  }
  
  return (
    <ContextMenu x={x} y={y} onClose={onClose}>
      <ContextMenuItem onClick={handleRename}>
        Rename Zone
      </ContextMenuItem>
      <ContextMenuSeparator />
      <ContextMenuItem onClick={handleDelete} danger>
        Delete Zone
      </ContextMenuItem>
    </ContextMenu>
  )
}
