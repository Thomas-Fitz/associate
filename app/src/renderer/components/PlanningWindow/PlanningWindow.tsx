import React, { useCallback, useMemo, useRef } from 'react'
import {
  ReactFlow,
  Controls,
  MiniMap,
  Background,
  BackgroundVariant,
  useNodesState,
  useEdgesState,
  SelectionMode,
  type Node,
  type Edge,
  type OnConnect,
  type OnNodesChange,
  type OnSelectionChangeFunc,
  type ReactFlowInstance,
  type NodeChange
} from '@xyflow/react'
import '@xyflow/react/dist/style.css'

import { TaskNode, type TaskNodeData, type TaskNodeType } from './TaskNode'
import { DependencyEdge, DependencyArrowMarker, type DependencyEdgeData, type DependencyEdgeType } from './DependencyEdge'
import { usePlans, useTasks, useSelection, useEdges } from '../../hooks'
import { useAppStore } from '../../stores/appStore'

const nodeTypes = { task: TaskNode }
const edgeTypes = { dependency: DependencyEdge }

export function PlanningWindow() {
  const { selectedPlan } = usePlans()
  const { 
    updateTaskContent, 
    updateTaskPosition, 
    updateTaskSize,
    createDependency 
  } = useTasks()
  const { selectedTaskIds, selectTasks, clearSelection, isSelected } = useSelection()
  const { selectedEdgeIds, selectEdges, clearEdgeSelection } = useEdges()
  const { showContextMenu } = useAppStore()
  
  const reactFlowRef = useRef<ReactFlowInstance<TaskNodeType, DependencyEdgeType> | null>(null)
  
  // Track if we're currently in a selection box drag to prevent clearing selection
  const isSelectingRef = useRef(false)
  
  // Handle content change
  const handleContentChange = useCallback(async (taskId: string, content: string) => {
    await updateTaskContent(taskId, { content })
  }, [updateTaskContent])
  
  // Handle size change
  const handleSizeChange = useCallback(async (taskId: string, width: number, height: number) => {
    await updateTaskSize(taskId, width, height)
  }, [updateTaskSize])
  
  // Handle task context menu
  const handleTaskContextMenu = useCallback((e: React.MouseEvent, taskId: string) => {
    showContextMenu(e.clientX, e.clientY, 'task', taskId)
  }, [showContextMenu])
  
  // Handle edge context menu - called by React Flow's onEdgeContextMenu
  const handleEdgeContextMenu = useCallback((e: React.MouseEvent, edge: DependencyEdgeType) => {
    e.preventDefault()
    showContextMenu(e.clientX, e.clientY, 'edge', undefined, undefined, undefined, edge.id)
  }, [showContextMenu])
  
  // Convert tasks to React Flow nodes
  // NOTE: We don't include isSelected here - selection state is managed separately
  // to avoid recreating nodes when selection changes (which would reset positions, etc.)
  const initialNodes = useMemo((): TaskNodeType[] => {
    if (!selectedPlan?.tasks) return []
    
    return selectedPlan.tasks.map((task, index) => ({
      id: task.id,
      type: 'task' as const,
      position: {
        x: task.metadata.ui_x ?? (index % 4) * 300 + 50,
        y: task.metadata.ui_y ?? Math.floor(index / 4) * 200 + 50
      },
      data: {
        task,
        isSelected: false, // Initial state, will be updated by selection effect
        onContentChange: handleContentChange,
        onSizeChange: handleSizeChange,
        onContextMenu: handleTaskContextMenu
      }
    }))
  }, [selectedPlan?.tasks, handleContentChange, handleSizeChange, handleTaskContextMenu])
  
  // Convert dependencies to React Flow edges
  const initialEdges = useMemo((): DependencyEdgeType[] => {
    if (!selectedPlan?.tasks) return []
    
    const edges: DependencyEdgeType[] = []
    
    for (const task of selectedPlan.tasks) {
      // Add DEPENDS_ON edges
      for (const depId of task.dependsOn) {
        edges.push({
          id: `${task.id}-depends-${depId}`,
          source: task.id,
          target: depId,
          type: 'dependency' as const,
          data: { relationshipType: 'DEPENDS_ON' as const }
        })
      }
      
      // Add BLOCKS edges
      for (const blockId of task.blocks) {
        edges.push({
          id: `${task.id}-blocks-${blockId}`,
          source: task.id,
          target: blockId,
          type: 'dependency' as const,
          data: { relationshipType: 'BLOCKS' as const }
        })
      }
    }
    
    return edges
  }, [selectedPlan?.tasks])
  
  const [nodes, setNodes, onNodesChange] = useNodesState<TaskNodeType>(initialNodes)
  const [edges, setEdges, onEdgesChange] = useEdgesState<DependencyEdgeType>(initialEdges)
  
  // Update nodes when plan changes
  React.useEffect(() => {
    setNodes(initialNodes)
  }, [initialNodes, setNodes])
  
  React.useEffect(() => {
    setEdges(initialEdges)
  }, [initialEdges, setEdges])
  
  // Update node selection state - sync both React Flow's 'selected' and our custom 'isSelected'
  // Skip during active selection box drag to avoid interfering with React Flow's selection
  React.useEffect(() => {
    if (isSelectingRef.current) {
      return
    }
    
    setNodes(nodes => nodes.map(node => {
      const nodeIsSelected = isSelected(node.id)
      return {
        ...node,
        selected: nodeIsSelected,
        data: {
          ...node.data,
          isSelected: nodeIsSelected
        }
      }
    }))
  }, [selectedTaskIds, setNodes, isSelected])
  
  // Update edge selection state
  React.useEffect(() => {
    if (isSelectingRef.current) {
      return
    }
    
    setEdges(edges => edges.map(edge => ({
      ...edge,
      selected: selectedEdgeIds.has(edge.id)
    })))
  }, [selectedEdgeIds, setEdges])
  
  // Handle node changes (position, selection, etc.)
  const handleNodesChange: OnNodesChange<TaskNodeType> = useCallback((changes) => {
    onNodesChange(changes)
    
    // Persist position changes
    for (const change of changes) {
      if (change.type === 'position' && 'position' in change && change.position && !change.dragging) {
        updateTaskPosition(change.id, change.position.x, change.position.y)
      }
    }
  }, [onNodesChange, updateTaskPosition])
  
  // Handle selection change from React Flow
  const handleSelectionChange: OnSelectionChangeFunc = useCallback(({ nodes, edges }) => {
    // During a selection box drag, ignore callbacks that would clear the selection
    // These can happen due to React re-renders causing spurious events
    if (isSelectingRef.current && nodes.length === 0) {
      return
    }
    
    const selectedNodeIds = nodes.map(n => n.id)
    selectTasks(selectedNodeIds)
    
    const selectedEdgeIdsList = edges.map(e => e.id)
    selectEdges(selectedEdgeIdsList)
  }, [selectTasks, selectEdges])
  
  // Handle connection (create dependency)
  const handleConnect: OnConnect = useCallback(async (connection) => {
    if (connection.source && connection.target) {
      try {
        await createDependency(connection.source, connection.target)
      } catch (err) {
        console.error('Failed to create dependency:', err)
        // TODO: Show error toast
      }
    }
  }, [createDependency])
  
  // Handle pane (canvas) context menu - only fires on empty canvas, not on nodes or edges
  const handlePaneContextMenu = useCallback((e: MouseEvent | React.MouseEvent) => {
    e.preventDefault()
    
    // Convert screen coordinates to canvas/flow coordinates
    if (reactFlowRef.current) {
      const canvasPosition = reactFlowRef.current.screenToFlowPosition({
        x: e.clientX,
        y: e.clientY
      })
      showContextMenu(e.clientX, e.clientY, 'canvas', undefined, canvasPosition.x, canvasPosition.y)
    } else {
      showContextMenu(e.clientX, e.clientY, 'canvas', undefined)
    }
  }, [showContextMenu])
  
  // Handle click on empty canvas
  const handlePaneClick = useCallback(() => {
    clearSelection()
    clearEdgeSelection()
  }, [clearSelection, clearEdgeSelection])
  
  // Track selection box start/end to prevent spurious empty selection events
  const handleSelectionStart = useCallback(() => {
    isSelectingRef.current = true
  }, [])
  
  const handleSelectionEnd = useCallback(() => {
    isSelectingRef.current = false
  }, [])
  
  if (!selectedPlan) {
    return (
      <div className="flex-1 flex items-center justify-center bg-surface-50">
        <div className="text-center text-surface-500">
          <div className="text-lg mb-2">No plan selected</div>
          <div className="text-sm">Select a plan from the sidebar to view tasks</div>
        </div>
      </div>
    )
  }
  
  return (
    <div className="flex-1 relative">
      <DependencyArrowMarker />
      
      <ReactFlow
        nodes={nodes}
        edges={edges}
        nodeTypes={nodeTypes}
        edgeTypes={edgeTypes}
        onNodesChange={handleNodesChange}
        onEdgesChange={onEdgesChange}
        onConnect={handleConnect}
        onSelectionChange={handleSelectionChange}
        onSelectionStart={handleSelectionStart}
        onSelectionEnd={handleSelectionEnd}
        onPaneClick={handlePaneClick}
        onPaneContextMenu={handlePaneContextMenu}
        onEdgeContextMenu={handleEdgeContextMenu}
        onInit={(instance) => { reactFlowRef.current = instance }}
        fitView
        minZoom={0.1}
        maxZoom={2}
        panOnScroll
        selectionOnDrag
        selectionMode={SelectionMode.Partial}
        multiSelectionKeyCode="Control"
        deleteKeyCode={null} // We handle delete via context menu
        selectNodesOnDrag={false}
        nodeDragThreshold={5}
        elevateEdgesOnSelect
        defaultEdgeOptions={{
          style: { strokeWidth: 2 },
          animated: false
        }}
      >
        <Controls />
        <MiniMap
          nodeColor={(node) => {
            const taskNode = node as TaskNodeType
            return taskNode.data?.isSelected ? '#0ea5e9' : '#e4e4e7'
          }}
          maskColor="rgba(0, 0, 0, 0.1)"
        />
        <Background variant={BackgroundVariant.Dots} gap={12} size={1} />
      </ReactFlow>
      
      {/* Plan header */}
      <div className="absolute top-4 left-4 bg-white/90 backdrop-blur-sm px-4 py-2 rounded-lg shadow-sm border border-surface-200">
        <div className="text-sm font-medium text-surface-800">{selectedPlan.name}</div>
        <div className="text-xs text-surface-500">{selectedPlan.tasks.length} tasks</div>
      </div>
    </div>
  )
}
