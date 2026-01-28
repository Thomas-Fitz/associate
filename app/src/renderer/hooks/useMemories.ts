import { useCallback } from 'react'
import { useAppStore } from '../stores/appStore'
import { useDatabase } from './useDatabase'
import type { MemoryInZone } from '../types'

export function useMemories() {
  const db = useDatabase()
  const {
    selectedZone,
    setSelectedZone,
    selectedMemoryIds,
    setSelectedMemoryIds,
    toggleMemorySelection,
    clearMemorySelection
  } = useAppStore()
  
  // Create a new memory in the selected zone
  const createMemory = useCallback(async (options: {
    type: 'Note' | 'Repository' | 'Memory'
    content: string
    metadata?: Record<string, unknown>
    tags?: string[]
  }): Promise<MemoryInZone | null> => {
    if (!selectedZone) {
      console.error('No zone selected')
      return null
    }
    
    try {
      const memory = await db.memories.create({
        zoneId: selectedZone.id,
        ...options
      })
      
      // Update local state
      if (selectedZone) {
        setSelectedZone({
          ...selectedZone,
          memories: [...selectedZone.memories, memory],
          memoryCount: (selectedZone.memoryCount || 0) + 1
        })
      }
      
      return memory
    } catch (err) {
      console.error('Failed to create memory:', err)
      throw err
    }
  }, [db.memories, selectedZone, setSelectedZone])
  
  // Update a memory
  const updateMemory = useCallback(async (memoryId: string, options: {
    content?: string
    metadata?: Record<string, unknown>
    tags?: string[]
  }): Promise<MemoryInZone | null> => {
    try {
      const updatedMemory = await db.memories.update(memoryId, options)
      
      // Update local state
      if (selectedZone) {
        setSelectedZone({
          ...selectedZone,
          memories: selectedZone.memories.map(m =>
            m.id === memoryId ? updatedMemory : m
          )
        })
      }
      
      return updatedMemory
    } catch (err) {
      console.error('Failed to update memory:', err)
      throw err
    }
  }, [db.memories, selectedZone, setSelectedZone])
  
  // Delete a memory
  const deleteMemory = useCallback(async (memoryId: string): Promise<void> => {
    try {
      await db.memories.delete(memoryId)
      
      // Update local state
      if (selectedZone) {
        const newSelection = new Set(selectedMemoryIds)
        newSelection.delete(memoryId)
        
        setSelectedZone({
          ...selectedZone,
          memories: selectedZone.memories.filter(m => m.id !== memoryId),
          memoryCount: Math.max(0, (selectedZone.memoryCount || 0) - 1)
        })
        setSelectedMemoryIds(newSelection)
      }
    } catch (err) {
      console.error('Failed to delete memory:', err)
      throw err
    }
  }, [db.memories, selectedZone, selectedMemoryIds, setSelectedZone, setSelectedMemoryIds])
  
  // Delete multiple memories
  const deleteMemories = useCallback(async (memoryIds: string[]): Promise<void> => {
    try {
      await Promise.all(memoryIds.map(id => db.memories.delete(id)))
      
      // Update local state
      if (selectedZone) {
        const idsSet = new Set(memoryIds)
        const newSelection = new Set(selectedMemoryIds)
        memoryIds.forEach(id => newSelection.delete(id))
        
        setSelectedZone({
          ...selectedZone,
          memories: selectedZone.memories.filter(m => !idsSet.has(m.id)),
          memoryCount: Math.max(0, (selectedZone.memoryCount || 0) - memoryIds.length)
        })
        setSelectedMemoryIds(newSelection)
      }
    } catch (err) {
      console.error('Failed to delete memories:', err)
      throw err
    }
  }, [db.memories, selectedZone, selectedMemoryIds, setSelectedZone, setSelectedMemoryIds])
  
  // Link a memory to another node
  const linkMemoryTo = useCallback(async (memoryId: string, targetId: string): Promise<void> => {
    try {
      await db.memories.linkTo(memoryId, targetId)
    } catch (err) {
      console.error('Failed to link memory:', err)
      throw err
    }
  }, [db.memories])
  
  // Unlink a memory from a node
  const unlinkMemoryFrom = useCallback(async (memoryId: string, targetId: string): Promise<void> => {
    try {
      await db.memories.unlinkFrom(memoryId, targetId)
    } catch (err) {
      console.error('Failed to unlink memory:', err)
      throw err
    }
  }, [db.memories])
  
  return {
    memories: selectedZone?.memories || [],
    selectedMemoryIds,
    toggleMemorySelection,
    clearMemorySelection,
    createMemory,
    updateMemory,
    deleteMemory,
    deleteMemories,
    linkMemoryTo,
    unlinkMemoryFrom
  }
}
