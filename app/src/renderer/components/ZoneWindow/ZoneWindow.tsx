import React, { useCallback, useMemo, useRef, useEffect } from 'react'
import {
  ReactFlow,
  Controls,
  MiniMap,
  Background,
  BackgroundVariant,
  useNodesState,
  useEdgesState,
  SelectionMode,
  type OnConnect,
  type OnNodesChange,
  type OnSelectionChangeFunc,
  type ReactFlowInstance,
  addEdge
} from '@xyflow/react'
import '@xyflow/react/dist/style.css'

import { PlanNode, type PlanNodeData, type PlanNodeType, PLAN_DEFAULT_WIDTH, PLAN_DEFAULT_HEIGHT } from './PlanNode'
import { MemoryNode, type MemoryNodeData, type MemoryNodeType } from './MemoryNode'
import { ZoneTaskNode, type ZoneTaskNodeData, type ZoneTaskNodeType } from './ZoneTaskNode'
import { DependencyEdge, DependencyArrowMarker, type DependencyEdgeType } from '../PlanningWindow/DependencyEdge'
import { useZones } from '../../hooks/useZones'
import { useAppStore } from '../../stores/appStore'
import { useDatabase } from '../../hooks/useDatabase'
import type { ZoneWithContents } from '../../types/zone'

// Custom node types for the zone
const nodeTypes = {
  plan: PlanNode,
  memory: MemoryNode,
  zoneTask: ZoneTaskNode
}

const edgeTypes = {
  dependency: DependencyEdge
}

// Combined node type for the zone
type ZoneNode = PlanNodeType | MemoryNodeType | ZoneTaskNodeType
type ZoneEdge = DependencyEdgeType

// Default dimensions
const MEMORY_DEFAULT_WIDTH = 200
const MEMORY_DEFAULT_HEIGHT = 120
const TASK_DEFAULT_WIDTH = 200
const TASK_DEFAULT_HEIGHT = 120

// Convert zone data to ReactFlow nodes (only called once on mount or zone change)
function createNodesFromZone(zone: ZoneWithContents): ZoneNode[] {
  const nodes: ZoneNode[] = []

  // Add Plan group nodes
  zone.plans.forEach((plan) => {
    const width = (plan.metadata.ui_width as number) || PLAN_DEFAULT_WIDTH
    const height = (plan.metadata.ui_height as number) || PLAN_DEFAULT_HEIGHT

    nodes.push({
      id: plan.id,
      type: 'plan',
      position: {
        x: (plan.metadata.ui_x as number) ?? 50,
        y: (plan.metadata.ui_y as number) ?? 50
      },
      style: { zIndex: 0 },
      data: {
        plan,
        isSelected: false,
        width,
        height,
        onContextMenu: undefined,
        onResize: undefined,
        onDescriptionChange: undefined
      }
    } as PlanNodeType)

    // Add tasks as children of this plan
    plan.tasks.forEach((task) => {
      const taskWidth = (task.metadata.ui_width as number) || TASK_DEFAULT_WIDTH
      const taskHeight = (task.metadata.ui_height as number) || TASK_DEFAULT_HEIGHT

      nodes.push({
        id: task.id,
        type: 'zoneTask',
        position: {
          x: (task.metadata.ui_x as number) ?? 180,
          y: (task.metadata.ui_y as number) ?? 60
        },
        parentId: plan.id,
        extent: undefined,
        expandParent: true,
        data: {
          task,
          planId: plan.id,
          isSelected: false,
          width: taskWidth,
          height: taskHeight,
          onContentChange: undefined,
          onSizeChange: undefined,
          onContextMenu: undefined
        }
      } as ZoneTaskNodeType)
    })
  })

  // Add Memory nodes
  zone.memories.forEach((memory) => {
    nodes.push({
      id: memory.id,
      type: 'memory',
      position: {
        x: memory.ui_x ?? 900,
        y: memory.ui_y ?? 50
      },
      data: {
        memory,
        isSelected: false,
        width: memory.ui_width || MEMORY_DEFAULT_WIDTH,
        height: memory.ui_height || MEMORY_DEFAULT_HEIGHT,
        onContentChange: undefined,
        onContextMenu: undefined,
        onResize: undefined
      }
    } as MemoryNodeType)
  })

  return nodes
}

// Convert zone data to ReactFlow edges
function createEdgesFromZone(zone: ZoneWithContents): ZoneEdge[] {
  const edges: ZoneEdge[] = []

  zone.plans.forEach((plan) => {
    plan.tasks.forEach((task) => {
      task.dependsOn.forEach((depId) => {
        edges.push({
          id: `${task.id}-depends-${depId}`,
          source: depId,
          target: task.id,
          type: 'dependency',
          data: { relationshipType: 'DEPENDS_ON' }
        })
      })

      task.blocks.forEach((blockId) => {
        edges.push({
          id: `${task.id}-blocks-${blockId}`,
          source: task.id,
          target: blockId,
          type: 'dependency',
          data: { relationshipType: 'BLOCKS' }
        })
      })
    })
  })

  return edges
}

export function ZoneWindow() {
  const { selectedZone, refreshSelectedZone } = useZones()
  const { showContextMenu } = useAppStore()
  const db = useDatabase()
  
  const [selectedNodeIds, setSelectedNodeIds] = React.useState<Set<string>>(new Set())
  const [selectedEdgeIds, setSelectedEdgeIds] = React.useState<Set<string>>(new Set())
  
  const reactFlowRef = useRef<ReactFlowInstance<ZoneNode, ZoneEdge> | null>(null)
  const isSelectingRef = useRef(false)

  // Create initial nodes and edges from zone data (only once per zone change)
  const initialNodes = useMemo(() => 
    selectedZone ? createNodesFromZone(selectedZone) : [], 
    [selectedZone]
  )
  const initialEdges = useMemo(() => 
    selectedZone ? createEdgesFromZone(selectedZone) : [], 
    [selectedZone]
  )

  const [nodes, setNodes, onNodesChange] = useNodesState<ZoneNode>(initialNodes)
  const [edges, setEdges, onEdgesChange] = useEdgesState<ZoneEdge>(initialEdges)

  // Reset nodes when zone changes
  useEffect(() => {
    setNodes(initialNodes)
    setEdges(initialEdges)
  }, [initialNodes, initialEdges, setNodes, setEdges])

  // --- Callbacks that update node data (passed to node components) ---

  const handlePlanResize = useCallback(async (planId: string, width: number, height: number) => {
    setNodes((nds) =>
      nds.map((node) => {
        if (node.id === planId && node.type === 'plan') {
          return {
            ...node,
            data: {
              ...node.data,
              width,
              height
            }
          } as PlanNodeType
        }
        return node
      })
    )
    
    // Persist to database
    try {
      await db.plans.update(planId, { 
        metadata: { ui_width: width, ui_height: height } 
      })
    } catch (err) {
      console.error('Failed to persist plan size:', err)
    }
  }, [setNodes, db.plans])

  const handlePlanDescriptionChange = useCallback(async (planId: string, description: string) => {
    setNodes((nds) =>
      nds.map((node) => {
        if (node.id === planId && node.type === 'plan') {
          const planData = node.data as PlanNodeData
          return {
            ...node,
            data: {
              ...planData,
              plan: {
                ...planData.plan,
                description
              }
            }
          } as PlanNodeType
        }
        return node
      })
    )
    
    // Persist to database
    try {
      await db.plans.update(planId, { description })
    } catch (err) {
      console.error('Failed to persist plan description:', err)
    }
  }, [setNodes, db.plans])

  const handlePlanContextMenu = useCallback((e: React.MouseEvent, planId: string) => {
    e.preventDefault()
    showContextMenu(e.clientX, e.clientY, 'plan', { planId })
  }, [showContextMenu])

  const handleTaskContentChange = useCallback(async (taskId: string, content: string) => {
    setNodes((nds) =>
      nds.map((node) => {
        if (node.id === taskId && node.type === 'zoneTask') {
          const taskData = node.data as ZoneTaskNodeData
          return {
            ...node,
            data: {
              ...taskData,
              task: {
                ...taskData.task,
                content
              }
            }
          } as ZoneTaskNodeType
        }
        return node
      })
    )
    
    // Persist to database
    try {
      await db.tasks.update(taskId, { content })
    } catch (err) {
      console.error('Failed to persist task content:', err)
    }
  }, [setNodes, db.tasks])

  const handleTaskSizeChange = useCallback(async (taskId: string, width: number, height: number) => {
    setNodes((nds) =>
      nds.map((node) => {
        if (node.id === taskId && node.type === 'zoneTask') {
          return {
            ...node,
            data: {
              ...node.data,
              width,
              height
            }
          } as ZoneTaskNodeType
        }
        return node
      })
    )
    
    // Persist to database
    try {
      await db.tasks.update(taskId, { 
        metadata: { ui_width: width, ui_height: height } 
      })
    } catch (err) {
      console.error('Failed to persist task size:', err)
    }
  }, [setNodes, db.tasks])

  const handleTaskContextMenu = useCallback((e: React.MouseEvent, taskId: string) => {
    e.preventDefault()
    showContextMenu(e.clientX, e.clientY, 'task', { taskId })
  }, [showContextMenu])

  const handleMemoryContentChange = useCallback(async (memoryId: string, content: string) => {
    setNodes((nds) =>
      nds.map((node) => {
        if (node.id === memoryId && node.type === 'memory') {
          const memData = node.data as MemoryNodeData
          return {
            ...node,
            data: {
              ...memData,
              memory: {
                ...memData.memory,
                content
              }
            }
          } as MemoryNodeType
        }
        return node
      })
    )
    
    // Persist to database
    try {
      await db.memories.update(memoryId, { content })
    } catch (err) {
      console.error('Failed to persist memory content:', err)
    }
  }, [setNodes, db.memories])

  const handleMemoryContextMenu = useCallback((e: React.MouseEvent, memoryId: string) => {
    e.preventDefault()
    showContextMenu(e.clientX, e.clientY, 'memory', { memoryId })
  }, [showContextMenu])

  const handleMemoryResize = useCallback(async (memoryId: string, width: number, height: number) => {
    setNodes((nds) =>
      nds.map((node) => {
        if (node.id === memoryId && node.type === 'memory') {
          return {
            ...node,
            data: {
              ...node.data,
              width,
              height
            }
          } as MemoryNodeType
        }
        return node
      })
    )
    
    // Persist to database
    try {
      await db.memories.update(memoryId, { 
        metadata: { ui_width: width, ui_height: height } 
      })
    } catch (err) {
      console.error('Failed to persist memory size:', err)
    }
  }, [setNodes, db.memories])

  // --- Attach callbacks to nodes ---
  // We need to update callbacks when they change (they reference setNodes)
  useEffect(() => {
    setNodes((nds) =>
      nds.map((node) => {
        if (node.type === 'plan') {
          return {
            ...node,
            data: {
              ...node.data,
              onContextMenu: handlePlanContextMenu,
              onResize: handlePlanResize,
              onDescriptionChange: handlePlanDescriptionChange
            }
          } as PlanNodeType
        }
        if (node.type === 'zoneTask') {
          return {
            ...node,
            data: {
              ...node.data,
              onContentChange: handleTaskContentChange,
              onSizeChange: handleTaskSizeChange,
              onContextMenu: handleTaskContextMenu
            }
          } as ZoneTaskNodeType
        }
        if (node.type === 'memory') {
          return {
            ...node,
            data: {
              ...node.data,
              onContentChange: handleMemoryContentChange,
              onContextMenu: handleMemoryContextMenu,
              onResize: handleMemoryResize
            }
          } as MemoryNodeType
        }
        return node
      })
    )
  }, [
    setNodes,
    handlePlanContextMenu,
    handlePlanResize,
    handlePlanDescriptionChange,
    handleTaskContentChange,
    handleTaskSizeChange,
    handleTaskContextMenu,
    handleMemoryContentChange,
    handleMemoryContextMenu,
    handleMemoryResize
  ])

  // --- Update selection state in node data ---
  useEffect(() => {
    if (isSelectingRef.current) return

    setNodes((nds) =>
      nds.map((node) => {
        const nodeIsSelected = selectedNodeIds.has(node.id)
        if (node.data.isSelected !== nodeIsSelected) {
          return {
            ...node,
            selected: nodeIsSelected,
            data: {
              ...node.data,
              isSelected: nodeIsSelected
            }
          } as ZoneNode
        }
        return node
      })
    )
  }, [selectedNodeIds, setNodes])

  useEffect(() => {
    if (isSelectingRef.current) return

    setEdges((eds) =>
      eds.map((edge) => ({
        ...edge,
        selected: selectedEdgeIds.has(edge.id)
      }))
    )
  }, [selectedEdgeIds, setEdges])

  // --- Hit-testing for re-parenting ---
  const getPlansAtPosition = useCallback((x: number, y: number): PlanNodeType[] => {
    if (!reactFlowRef.current) return []
    
    const allNodes = reactFlowRef.current.getNodes()
    const planNodes = allNodes.filter((n): n is PlanNodeType => n.type === 'plan')
    
    return planNodes.filter((plan) => {
      const planData = plan.data as PlanNodeData
      const planWidth = planData.width || PLAN_DEFAULT_WIDTH
      const planHeight = planData.height || PLAN_DEFAULT_HEIGHT
      
      return (
        x >= plan.position.x &&
        x <= plan.position.x + planWidth &&
        y >= plan.position.y &&
        y <= plan.position.y + planHeight
      )
    })
  }, [])

  // --- Handle task re-parenting when dragged to a different plan ---
  const handleNodeDragStop = useCallback(
    async (_event: React.MouseEvent, node: ZoneNode, _nodes: ZoneNode[]) => {
      // Persist position for any node type
      const nodeType = node.type
      const nodeId = node.id
      const position = node.position
      
      try {
        if (nodeType === 'plan') {
          await db.plans.update(nodeId, { 
            metadata: { ui_x: position.x, ui_y: position.y } 
          })
        } else if (nodeType === 'zoneTask') {
          await db.tasks.update(nodeId, { 
            metadata: { ui_x: position.x, ui_y: position.y } 
          })
        } else if (nodeType === 'memory') {
          await db.memories.update(nodeId, { 
            metadata: { ui_x: position.x, ui_y: position.y } 
          })
        }
      } catch (err) {
        console.error('Failed to persist position:', err)
      }
      
      // Handle task re-parenting
      if (node.type !== 'zoneTask') return
      
      const taskNode = node as ZoneTaskNodeType
      const currentParentId = taskNode.parentId
      
      // Calculate absolute position
      let absoluteX = taskNode.position.x
      let absoluteY = taskNode.position.y
      
      if (currentParentId && reactFlowRef.current) {
        const parentNode = reactFlowRef.current.getNode(currentParentId)
        if (parentNode) {
          absoluteX += parentNode.position.x
          absoluteY += parentNode.position.y
        }
      }
      
      const centerX = absoluteX + 100
      const centerY = absoluteY + 60
      const plansAtPosition = getPlansAtPosition(centerX, centerY)
      const newParentCandidates = plansAtPosition.filter((p) => p.id !== currentParentId)
      
      if (newParentCandidates.length > 0) {
        const newParent = newParentCandidates[0]
        console.log(`Re-parenting task ${taskNode.id} from ${currentParentId || 'none'} to ${newParent.id}`)
        
        const newRelativeX = absoluteX - newParent.position.x
        const newRelativeY = absoluteY - newParent.position.y
        
        setNodes((currentNodes) =>
          currentNodes.map((n) => {
            if (n.id === taskNode.id) {
              return {
                ...n,
                parentId: newParent.id,
                position: { x: newRelativeX, y: newRelativeY },
                data: {
                  ...n.data,
                  planId: newParent.id
                }
              } as ZoneTaskNodeType
            }
            return n
          })
        )
        
        // TODO: Persist re-parenting to database (move task to new plan)
        // This would require a db.tasks.move(taskId, newPlanId) API
      }
    },
    [getPlansAtPosition, setNodes, db.plans, db.tasks, db.memories]
  )

  // --- Handle node changes ---
  const handleNodesChange: OnNodesChange<ZoneNode> = useCallback(
    (changes) => {
      onNodesChange(changes)
    },
    [onNodesChange]
  )

  // --- Handle selection change ---
  const handleSelectionChange: OnSelectionChangeFunc = useCallback(
    ({ nodes, edges }) => {
      if (isSelectingRef.current && nodes.length === 0) return

      setSelectedNodeIds(new Set(nodes.map((n) => n.id)))
      setSelectedEdgeIds(new Set(edges.map((e) => e.id)))
    },
    []
  )

  // --- Handle connection ---
  const handleConnect: OnConnect = useCallback(
    async (connection) => {
      if (connection.source && connection.target) {
        // Add edge to local state immediately
        setEdges((eds) =>
          addEdge(
            {
              ...connection,
              id: `${connection.target}-depends-${connection.source}`,
              type: 'dependency',
              data: { relationshipType: 'DEPENDS_ON' }
            },
            eds
          )
        )
        
        // Persist to database
        try {
          await db.dependencies.create(connection.target, connection.source)
        } catch (err) {
          console.error('Failed to create dependency:', err)
          // Remove edge on failure
          setEdges((eds) => eds.filter(e => 
            !(e.source === connection.source && e.target === connection.target)
          ))
        }
      }
    },
    [setEdges, db.dependencies]
  )

  // --- Handle canvas context menu ---
  const handlePaneContextMenu = useCallback((e: MouseEvent | React.MouseEvent) => {
    e.preventDefault()
    
    if (reactFlowRef.current) {
      const canvasPosition = reactFlowRef.current.screenToFlowPosition({
        x: e.clientX,
        y: e.clientY
      })
      showContextMenu(e.clientX, e.clientY, 'canvas', { 
        canvasX: canvasPosition.x, 
        canvasY: canvasPosition.y 
      })
    } else {
      showContextMenu(e.clientX, e.clientY, 'canvas', {})
    }
  }, [showContextMenu])

  const handlePaneClick = useCallback(() => {
    setSelectedNodeIds(new Set())
    setSelectedEdgeIds(new Set())
  }, [])

  const handleSelectionStart = useCallback(() => {
    isSelectingRef.current = true
  }, [])

  const handleSelectionEnd = useCallback(() => {
    isSelectingRef.current = false
  }, [])

  // --- Handle edge context menu ---
  const handleEdgeContextMenu = useCallback((e: React.MouseEvent, edge: ZoneEdge) => {
    e.preventDefault()
    showContextMenu(e.clientX, e.clientY, 'edge', { edgeId: edge.id })
  }, [showContextMenu])

  // Empty state when no zone is selected
  if (!selectedZone) {
    return (
      <div className="flex-1 flex items-center justify-center bg-surface-50">
        <div className="text-center text-surface-500">
          <div className="text-lg mb-2">No zone selected</div>
          <div className="text-sm">Select a zone from the sidebar to view its contents</div>
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
        onNodeDragStop={handleNodeDragStop}
        onPaneClick={handlePaneClick}
        onPaneContextMenu={handlePaneContextMenu}
        onEdgeContextMenu={handleEdgeContextMenu}
        onInit={(instance) => {
          reactFlowRef.current = instance
        }}
        fitView
        minZoom={0.1}
        maxZoom={2}
        panOnScroll
        selectionOnDrag
        selectionMode={SelectionMode.Partial}
        multiSelectionKeyCode="Control"
        deleteKeyCode={null}
        selectNodesOnDrag={false}
        nodeDragThreshold={5}
        elevateEdgesOnSelect
        nodesDraggable
        connectOnClick
        defaultEdgeOptions={{
          style: { strokeWidth: 2 },
          animated: false
        }}
      >
        <Controls />
        <MiniMap
          nodeColor={(node) => {
            if (node.type === 'plan') return '#e0e7ff'
            if (node.type === 'memory') return '#fef3c7'
            return node.selected ? '#0ea5e9' : '#e4e4e7'
          }}
          maskColor="rgba(0, 0, 0, 0.1)"
          pannable
          zoomable
        />
        <Background variant={BackgroundVariant.Dots} gap={12} size={1} />
      </ReactFlow>

      {/* Zone header */}
      <div className="absolute top-4 left-4 bg-white/90 backdrop-blur-sm px-4 py-2 rounded-lg shadow-sm border border-surface-200">
        <div className="text-sm font-medium text-surface-800">{selectedZone.name}</div>
        <div className="text-xs text-surface-500">
          {selectedZone.plans.length} plans | {selectedZone.plans.reduce((acc, p) => acc + p.tasks.length, 0)} tasks | {selectedZone.memories.length} memories
        </div>
      </div>
    </div>
  )
}
