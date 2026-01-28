import { useCallback, useEffect } from 'react'
import { useAppStore } from '../stores/appStore'
import { useDatabase } from './useDatabase'

export function useZones() {
  const db = useDatabase()
  const {
    zones,
    selectedZoneId,
    selectedZone,
    zonesLoading,
    zonesError,
    searchQuery,
    setZones,
    setSelectedZoneId,
    setSelectedZone,
    setZonesLoading,
    setZonesError
  } = useAppStore()
  
  // Load zones list
  const loadZones = useCallback(async () => {
    setZonesLoading(true)
    setZonesError(null)
    
    try {
      const options: { search?: string } = {}
      
      if (searchQuery.trim()) {
        options.search = searchQuery.trim()
      }
      
      const loadedZones = await db.zones.list(options)
      setZones(loadedZones)
    } catch (err) {
      console.error('Failed to load zones:', err)
      setZonesError(err instanceof Error ? err.message : 'Failed to load zones')
    } finally {
      setZonesLoading(false)
    }
  }, [db.zones, searchQuery, setZones, setZonesLoading, setZonesError])
  
  // Load selected zone with contents
  const loadSelectedZone = useCallback(async (zoneId: string) => {
    try {
      const zone = await db.zones.get(zoneId)
      setSelectedZone(zone)
    } catch (err) {
      console.error('Failed to load zone:', err)
      setSelectedZone(null)
    }
  }, [db.zones, setSelectedZone])
  
  // Select a zone
  const selectZone = useCallback((zoneId: string | null) => {
    setSelectedZoneId(zoneId)
    if (zoneId) {
      loadSelectedZone(zoneId)
    } else {
      setSelectedZone(null)
    }
  }, [setSelectedZoneId, setSelectedZone, loadSelectedZone])
  
  // Create a new zone
  const createZone = useCallback(async (options: { name: string; description?: string }) => {
    try {
      const zone = await db.zones.create(options)
      await loadZones()
      return zone
    } catch (err) {
      console.error('Failed to create zone:', err)
      throw err
    }
  }, [db.zones, loadZones])
  
  // Rename a zone
  const renameZone = useCallback(async (zoneId: string, name: string) => {
    try {
      await db.zones.update(zoneId, { name })
      await loadZones()
      if (selectedZoneId === zoneId) {
        await loadSelectedZone(zoneId)
      }
    } catch (err) {
      console.error('Failed to rename zone:', err)
      throw err
    }
  }, [db.zones, loadZones, selectedZoneId, loadSelectedZone])
  
  // Delete a zone
  const deleteZone = useCallback(async (zoneId: string) => {
    try {
      await db.zones.delete(zoneId)
      if (selectedZoneId === zoneId) {
        setSelectedZoneId(null)
        setSelectedZone(null)
      }
      await loadZones()
    } catch (err) {
      console.error('Failed to delete zone:', err)
      throw err
    }
  }, [db.zones, selectedZoneId, setSelectedZoneId, setSelectedZone, loadZones])
  
  // Load zones on mount and when search changes
  useEffect(() => {
    loadZones()
  }, [loadZones])
  
  return {
    zones,
    selectedZoneId,
    selectedZone,
    loading: zonesLoading,
    error: zonesError,
    selectZone,
    createZone,
    renameZone,
    deleteZone,
    refreshZones: loadZones,
    refreshSelectedZone: () => selectedZoneId && loadSelectedZone(selectedZoneId)
  }
}
