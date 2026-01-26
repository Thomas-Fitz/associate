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
import { usePlans, useTasks, useSelection } from '../../hooks'
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
  const { showContextMenu } = useAppStore()
  
  const reactFlowRef = useRef<ReactFlowInstance<TaskNodeType, DependencyEdgeType> | null>(null)
  
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
  
  // Convert tasks to React Flow nodes
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
        isSelected: isSelected(task.id),
        onContentChange: handleContentChange,
        onSizeChange: handleSizeChange,
        onContextMenu: handleTaskContextMenu
      }
    }))
  }, [selectedPlan?.tasks, isSelected, handleContentChange, handleSizeChange, handleTaskContextMenu])
  
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
  
  // Update node selection state
  React.useEffect(() => {
    setNodes(nodes => nodes.map(node => ({
      ...node,
      data: {
        ...node.data,
        isSelected: isSelected(node.id)
      }
    })))
  }, [selectedTaskIds, setNodes, isSelected])
  
  // Handle node position change (drag end)
  const handleNodesChange: OnNodesChange<TaskNodeType> = useCallback((changes) => {
    onNodesChange(changes)
    
    // Persist position changes
    for (const change of changes) {
      if (change.type === 'position' && 'position' in change && change.position && !change.dragging) {
        updateTaskPosition(change.id, change.position.x, change.position.y)
      }
    }
  }, [onNodesChange, updateTaskPosition])
  
  // Handle selection change
  const handleSelectionChange: OnSelectionChangeFunc = useCallback(({ nodes }) => {
    const selectedIds = nodes.map(n => n.id)
    selectTasks(selectedIds)
  }, [selectTasks])
  
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
  
  // Handle canvas context menu
  const handleCanvasContextMenu = useCallback((e: React.MouseEvent) => {
    e.preventDefault()
    
    // Get canvas position
    if (reactFlowRef.current) {
      showContextMenu(e.clientX, e.clientY, 'canvas', undefined)
    }
  }, [showContextMenu])
  
  // Handle click on empty canvas
  const handlePaneClick = useCallback(() => {
    clearSelection()
  }, [clearSelection])
  
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
    <div className="flex-1 relative" onContextMenu={handleCanvasContextMenu}>
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
        onPaneClick={handlePaneClick}
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
