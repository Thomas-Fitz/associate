import React, { useState } from 'react'
import { ZoneSearch } from './ZoneSearch'
import { ZoneList } from './ZoneList'
import { useZones } from '../../hooks'
import { useAppStore } from '../../stores/appStore'

export function Sidebar() {
  const { zones, selectedZoneId, selectZone, createZone, renameZone, deleteZone, loading, error } = useZones()
  const { showDeleteZoneDialog } = useAppStore()
  const [isCreating, setIsCreating] = useState(false)
  const [newZoneName, setNewZoneName] = useState('')

  const handleCreateZone = async () => {
    if (!newZoneName.trim()) return
    
    try {
      const zone = await createZone({ name: newZoneName.trim() })
      setNewZoneName('')
      setIsCreating(false)
      if (zone) {
        selectZone(zone.id)
      }
    } catch (err) {
      console.error('Failed to create zone:', err)
    }
  }

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter') {
      handleCreateZone()
    } else if (e.key === 'Escape') {
      setIsCreating(false)
      setNewZoneName('')
    }
  }

  const handleDeleteZone = (zoneId: string) => {
    const zone = zones.find(z => z.id === zoneId)
    if (zone) {
      showDeleteZoneDialog(zoneId, zone.name)
    }
  }
  
  return (
    <aside 
      className="w-sidebar flex flex-col bg-white border-r border-surface-200 h-full"
      aria-label="Sidebar"
    >
      {/* Header */}
      <div className="p-3 border-b border-surface-200 flex items-center justify-between">
        <h1 className="text-lg font-semibold text-surface-800">Zones</h1>
        <button
          onClick={() => setIsCreating(true)}
          className="p-1.5 text-surface-500 hover:text-primary-600 hover:bg-primary-50 rounded-md transition-colors"
          title="Create new zone"
          aria-label="Create new zone"
        >
          <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4v16m8-8H4" />
          </svg>
        </button>
      </div>

      {/* Create Zone Input */}
      {isCreating && (
        <div className="p-3 border-b border-surface-200 bg-primary-50">
          <input
            type="text"
            placeholder="Zone name..."
            value={newZoneName}
            onChange={(e) => setNewZoneName(e.target.value)}
            onKeyDown={handleKeyDown}
            onBlur={() => {
              if (!newZoneName.trim()) {
                setIsCreating(false)
              }
            }}
            autoFocus
            className="w-full px-3 py-2 text-sm border border-primary-300 rounded-md 
                       focus:outline-none focus:ring-2 focus:ring-primary-500 focus:border-transparent
                       placeholder:text-surface-400"
          />
          <div className="flex gap-2 mt-2">
            <button
              onClick={handleCreateZone}
              disabled={!newZoneName.trim()}
              className="flex-1 px-3 py-1.5 text-sm bg-primary-500 text-white rounded-md 
                         hover:bg-primary-600 disabled:opacity-50 disabled:cursor-not-allowed"
            >
              Create
            </button>
            <button
              onClick={() => {
                setIsCreating(false)
                setNewZoneName('')
              }}
              className="px-3 py-1.5 text-sm border border-surface-300 rounded-md hover:bg-surface-50"
            >
              Cancel
            </button>
          </div>
        </div>
      )}
      
      {/* Search */}
      <ZoneSearch />
      
      {/* Zone List */}
      <ZoneList
        zones={zones}
        selectedZoneId={selectedZoneId}
        onSelectZone={selectZone}
        onRenameZone={renameZone}
        onDeleteZone={handleDeleteZone}
        loading={loading}
        error={error}
      />
    </aside>
  )
}
