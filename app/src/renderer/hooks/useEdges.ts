import { useCallback } from 'react'
import { useAppStore } from '../stores/appStore'
import { useTasks } from './useTasks'
import type { EdgeInfo, RelationshipType } from '../types'

/**
 * Parse edge info from a React Flow edge ID and data
 * Edge IDs follow the format: {sourceTaskId}-depends-{targetTaskId} or {sourceTaskId}-blocks-{targetTaskId}
 */
export function parseEdgeInfo(edgeId: string, relationshipType?: RelationshipType): EdgeInfo | null {
  // Try to parse DEPENDS_ON format: {sourceTaskId}-depends-{targetTaskId}
  const dependsMatch = edgeId.match(/^(.+)-depends-(.+)$/)
  if (dependsMatch) {
    return {
      id: edgeId,
      sourceTaskId: dependsMatch[1],
      targetTaskId: dependsMatch[2],
      relationshipType: relationshipType || 'DEPENDS_ON'
    }
  }
  
  // Try to parse BLOCKS format: {sourceTaskId}-blocks-{targetTaskId}
  const blocksMatch = edgeId.match(/^(.+)-blocks-(.+)$/)
  if (blocksMatch) {
    return {
      id: edgeId,
      sourceTaskId: blocksMatch[1],
      targetTaskId: blocksMatch[2],
      relationshipType: relationshipType || 'BLOCKS'
    }
  }
  
  return null
}

export function useEdges() {
  const {
    selectedEdgeIds,
    setSelectedEdgeIds,
    clearEdgeSelection,
    showDeleteEdgeDialog,
    hideDeleteEdgeDialog,
    selectedPlan
  } = useAppStore()
  
  const { deleteDependency } = useTasks()
  
  // Select a single edge (clear others)
  const selectEdge = useCallback((edgeId: string) => {
    setSelectedEdgeIds(new Set([edgeId]))
  }, [setSelectedEdgeIds])
  
  // Select multiple edges (replace selection)
  const selectEdges = useCallback((edgeIds: string[]) => {
    setSelectedEdgeIds(new Set(edgeIds))
  }, [setSelectedEdgeIds])
  
  // Check if an edge is selected
  const isEdgeSelected = useCallback((edgeId: string) => {
    return selectedEdgeIds.has(edgeId)
  }, [selectedEdgeIds])
  
  // Get selected edge IDs as array
  const getSelectedEdgeIds = useCallback(() => {
    return Array.from(selectedEdgeIds)
  }, [selectedEdgeIds])
  
  // Get EdgeInfo for an edge ID
  const getEdgeInfo = useCallback((edgeId: string): EdgeInfo | null => {
    return parseEdgeInfo(edgeId)
  }, [])
  
  // Get EdgeInfo for multiple edge IDs
  const getEdgesInfo = useCallback((edgeIds: string[]): EdgeInfo[] => {
    return edgeIds.map(id => parseEdgeInfo(id)).filter((e): e is EdgeInfo => e !== null)
  }, [])
  
  // Show confirmation dialog for edge deletion
  const confirmDeleteEdges = useCallback((edgeIds?: string[]) => {
    const ids = edgeIds || Array.from(selectedEdgeIds)
    if (ids.length === 0) return
    
    const edges = getEdgesInfo(ids)
    if (edges.length === 0) return
    
    showDeleteEdgeDialog(edges)
  }, [selectedEdgeIds, getEdgesInfo, showDeleteEdgeDialog])
  
  // Delete edges (called after confirmation)
  const deleteEdges = useCallback(async (edges: EdgeInfo[]) => {
    await Promise.all(
      edges.map(edge => 
        deleteDependency(edge.sourceTaskId, edge.targetTaskId, edge.relationshipType)
      )
    )
    
    // Clear selection of deleted edges
    const deletedIds = new Set(edges.map(e => e.id))
    const newSelection = new Set(
      Array.from(selectedEdgeIds).filter(id => !deletedIds.has(id))
    )
    setSelectedEdgeIds(newSelection)
    
    hideDeleteEdgeDialog()
  }, [deleteDependency, selectedEdgeIds, setSelectedEdgeIds, hideDeleteEdgeDialog])
  
  // Get description for an edge (for UI display)
  const getEdgeDescription = useCallback((edge: EdgeInfo): string => {
    if (!selectedPlan) return ''
    
    const sourceTask = selectedPlan.tasks.find(t => t.id === edge.sourceTaskId)
    const targetTask = selectedPlan.tasks.find(t => t.id === edge.targetTaskId)
    
    const sourceLabel = sourceTask 
      ? (sourceTask.content.length > 30 ? sourceTask.content.substring(0, 30) + '...' : sourceTask.content)
      : edge.sourceTaskId
    const targetLabel = targetTask
      ? (targetTask.content.length > 30 ? targetTask.content.substring(0, 30) + '...' : targetTask.content)
      : edge.targetTaskId
    
    const relationLabel = edge.relationshipType === 'DEPENDS_ON' ? 'depends on' : 'blocks'
    
    return `"${sourceLabel}" ${relationLabel} "${targetLabel}"`
  }, [selectedPlan])
  
  return {
    selectedEdgeIds,
    selectedEdgeCount: selectedEdgeIds.size,
    selectEdge,
    selectEdges,
    clearEdgeSelection,
    isEdgeSelected,
    getSelectedEdgeIds,
    getEdgeInfo,
    getEdgesInfo,
    confirmDeleteEdges,
    deleteEdges,
    getEdgeDescription
  }
}
