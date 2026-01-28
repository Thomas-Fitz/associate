import React, { useState, useRef, useEffect } from 'react'
import type { Zone } from '../../types'
import { useAppStore } from '../../stores/appStore'

interface ZoneListProps {
  zones: Zone[]
  selectedZoneId: string | null
  onSelectZone: (zoneId: string) => void
  onRenameZone: (zoneId: string, name: string) => Promise<void>
  onDeleteZone: (zoneId: string) => void
  loading?: boolean
  error?: string | null
}

export function ZoneList({ 
  zones, 
  selectedZoneId, 
  onSelectZone, 
  onRenameZone,
  onDeleteZone,
  loading, 
  error 
}: ZoneListProps) {
  const { showContextMenu } = useAppStore()
  const [editingZoneId, setEditingZoneId] = useState<string | null>(null)
  const [editingName, setEditingName] = useState('')
  const inputRef = useRef<HTMLInputElement>(null)

  // Focus input when editing starts
  useEffect(() => {
    if (editingZoneId && inputRef.current) {
      inputRef.current.focus()
      inputRef.current.select()
    }
  }, [editingZoneId])

  const handleContextMenu = (e: React.MouseEvent, zone: Zone) => {
    e.preventDefault()
    showContextMenu(e.clientX, e.clientY, 'zone', { zoneId: zone.id })
  }

  const handleDoubleClick = (zone: Zone) => {
    setEditingZoneId(zone.id)
    setEditingName(zone.name)
  }

  const handleRenameSubmit = async (zoneId: string) => {
    if (editingName.trim() && editingName !== zones.find(z => z.id === zoneId)?.name) {
      try {
        await onRenameZone(zoneId, editingName.trim())
      } catch (err) {
        console.error('Failed to rename zone:', err)
      }
    }
    setEditingZoneId(null)
    setEditingName('')
  }

  const handleKeyDown = (e: React.KeyboardEvent, zoneId: string) => {
    if (e.key === 'Enter') {
      handleRenameSubmit(zoneId)
    } else if (e.key === 'Escape') {
      setEditingZoneId(null)
      setEditingName('')
    }
  }

  if (loading) {
    return (
      <div className="flex-1 flex items-center justify-center">
        <div className="text-surface-500 text-sm">Loading zones...</div>
      </div>
    )
  }
  
  if (error) {
    return (
      <div className="flex-1 p-4">
        <div className="text-red-600 text-sm">{error}</div>
      </div>
    )
  }
  
  if (zones.length === 0) {
    return (
      <div className="flex-1 flex items-center justify-center">
        <div className="text-surface-500 text-sm text-center px-4">
          <p>No zones found</p>
          <p className="text-xs mt-1">Create a zone to get started</p>
        </div>
      </div>
    )
  }
  
  return (
    <div className="flex-1 overflow-y-auto" role="listbox" aria-label="Zones">
      {zones.map((zone) => (
        <button
          key={zone.id}
          onClick={() => onSelectZone(zone.id)}
          onContextMenu={(e) => handleContextMenu(e, zone)}
          onDoubleClick={() => handleDoubleClick(zone)}
          className={`w-full text-left p-3 border-b border-surface-100 hover:bg-surface-100 
                     transition-colors cursor-pointer ${
                       selectedZoneId === zone.id ? 'bg-primary-50 border-l-4 border-l-primary-500' : ''
                     }`}
          role="option"
          aria-selected={selectedZoneId === zone.id}
        >
          <div className="flex items-start justify-between gap-2">
            <div className="flex-1 min-w-0">
              {editingZoneId === zone.id ? (
                <input
                  ref={inputRef}
                  type="text"
                  value={editingName}
                  onChange={(e) => setEditingName(e.target.value)}
                  onBlur={() => handleRenameSubmit(zone.id)}
                  onKeyDown={(e) => handleKeyDown(e, zone.id)}
                  onClick={(e) => e.stopPropagation()}
                  className="w-full px-1 py-0.5 text-sm font-medium border border-primary-500 rounded
                             focus:outline-none focus:ring-1 focus:ring-primary-500"
                />
              ) : (
                <div className="font-medium text-sm truncate" title={zone.name}>
                  {zone.name}
                </div>
              )}
              {zone.description && !editingZoneId && (
                <div className="text-xs text-surface-500 truncate mt-0.5" title={zone.description}>
                  {zone.description}
                </div>
              )}
            </div>
          </div>
          <div className="text-xs text-surface-400 mt-1">
            {zone.planCount ?? 0} plan{(zone.planCount ?? 0) !== 1 ? 's' : ''} | {' '}
            {zone.taskCount ?? 0} task{(zone.taskCount ?? 0) !== 1 ? 's' : ''}
            {(zone.memoryCount ?? 0) > 0 && (
              <> | {zone.memoryCount} memor{(zone.memoryCount ?? 0) !== 1 ? 'ies' : 'y'}</>
            )}
          </div>
        </button>
      ))}
    </div>
  )
}
