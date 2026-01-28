import React, { useCallback, useMemo, useRef, useState, useEffect } from 'react'
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
  addEdge
} from '@xyflow/react'
import '@xyflow/react/dist/style.css'

import { PlanNode, type PlanNodeData, type PlanNodeType, PLAN_DEFAULT_WIDTH, PLAN_DEFAULT_HEIGHT } from './PlanNode'
import { MemoryNode, type MemoryNodeData, type MemoryNodeType } from './MemoryNode'
import { ZoneTaskNode, type ZoneTaskNodeData, type ZoneTaskNodeType } from './ZoneTaskNode'
import { DependencyEdge, DependencyArrowMarker, type DependencyEdgeType } from '../PlanningWindow/DependencyEdge'
import type { ZoneWithContents, TaskInZone } from '../../types/zone'

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

// Mock data for Phase 0 prototype testing
function createMockZone(): ZoneWithContents {
  const now = new Date().toISOString()
  
  return {
    id: 'zone-1',
    name: 'Sprint 12 - User Auth',
    description: 'Zone for user authentication feature work',
    metadata: {},
    tags: ['sprint-12', 'auth'],
    createdAt: now,
    updatedAt: now,
    planCount: 2,
    taskCount: 5,
    memoryCount: 2,
    plans: [
      {
        id: 'plan-1',
        name: 'Backend Auth',
        description: 'Implement backend authentication including JWT tokens, session management, and secure password handling. This plan covers all server-side auth logic.',
        status: 'active',
        metadata: { ui_x: 50, ui_y: 50, ui_width: 500, ui_height: 380 },
        tags: ['backend'],
        createdAt: now,
        updatedAt: now,
        tasks: [
          {
            id: 'task-1',
            content: 'Set up JWT token generation',
            status: 'completed',
            metadata: { ui_x: 180, ui_y: 60, ui_width: 200, ui_height: 100 },
            tags: [],
            createdAt: now,
            updatedAt: now,
            planId: 'plan-1',
            dependsOn: [],
            blocks: ['task-2']
          },
          {
            id: 'task-2',
            content: 'Implement refresh token logic',
            status: 'in_progress',
            metadata: { ui_x: 180, ui_y: 200, ui_width: 200, ui_height: 100 },
            tags: [],
            createdAt: now,
            updatedAt: now,
            planId: 'plan-1',
            dependsOn: ['task-1'],
            blocks: []
          }
        ]
      },
      {
        id: 'plan-2',
        name: 'Frontend Auth',
        description: 'Implement frontend authentication UI and state management. Includes login/logout forms, protected routes, and token storage.',
        status: 'draft',
        metadata: { ui_x: 600, ui_y: 50, ui_width: 500, ui_height: 420 },
        tags: ['frontend'],
        createdAt: now,
        updatedAt: now,
        tasks: [
          {
            id: 'task-3',
            content: 'Create login form component',
            status: 'pending',
            metadata: { ui_x: 180, ui_y: 60, ui_width: 200, ui_height: 100 },
            tags: [],
            createdAt: now,
            updatedAt: now,
            planId: 'plan-2',
            dependsOn: [],
            blocks: ['task-4']
          },
          {
            id: 'task-4',
            content: 'Add form validation',
            status: 'pending',
            metadata: { ui_x: 180, ui_y: 200, ui_width: 200, ui_height: 100 },
            tags: [],
            createdAt: now,
            updatedAt: now,
            planId: 'plan-2',
            dependsOn: ['task-3', 'task-2'],
            blocks: []
          },
          {
            id: 'task-5',
            content: 'Integrate with auth API',
            status: 'blocked',
            metadata: { ui_x: 180, ui_y: 320, ui_width: 200, ui_height: 100 },
            tags: [],
            createdAt: now,
            updatedAt: now,
            planId: 'plan-2',
            dependsOn: ['task-2'],
            blocks: []
          }
        ]
      }
    ],
    memories: [
      {
        id: 'memory-1',
        type: 'Note',
        content: 'Remember: Use httpOnly cookies for refresh tokens to prevent XSS attacks.',
        metadata: {},
        tags: ['security'],
        createdAt: now,
        updatedAt: now,
        ui_x: 1150,
        ui_y: 50,
        ui_width: 250,
        ui_height: 140
      },
      {
        id: 'memory-2',
        type: 'Repository',
        content: 'Auth reference: github.com/example/auth-patterns',
        metadata: {},
        tags: ['reference'],
        createdAt: now,
        updatedAt: now,
        ui_x: 1150,
        ui_y: 220,
        ui_width: 250,
        ui_height: 100
      }
    ]
  }
}

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
  const [zone] = useState<ZoneWithContents>(createMockZone)
  const [selectedNodeIds, setSelectedNodeIds] = useState<Set<string>>(new Set())
  const [selectedEdgeIds, setSelectedEdgeIds] = useState<Set<string>>(new Set())
  
  const reactFlowRef = useRef<ReactFlowInstance<ZoneNode, ZoneEdge> | null>(null)
  const isSelectingRef = useRef(false)

  // Create initial nodes and edges from zone data (only once)
  const initialNodes = useMemo(() => createNodesFromZone(zone), [zone])
  const initialEdges = useMemo(() => createEdgesFromZone(zone), [zone])

  const [nodes, setNodes, onNodesChange] = useNodesState<ZoneNode>(initialNodes)
  const [edges, setEdges, onEdgesChange] = useEdgesState<ZoneEdge>(initialEdges)

  // --- Callbacks that update node data (passed to node components) ---

  const handlePlanResize = useCallback((planId: string, width: number, height: number) => {
    console.log('Plan resized:', planId, width, height)
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
  }, [setNodes])

  const handlePlanDescriptionChange = useCallback((planId: string, description: string) => {
    console.log('Plan description changed:', planId, description)
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
  }, [setNodes])

  const handlePlanContextMenu = useCallback((e: React.MouseEvent, planId: string) => {
    console.log('Plan context menu:', planId)
  }, [])

  const handleTaskContentChange = useCallback((taskId: string, content: string) => {
    console.log('Task content changed:', taskId, content)
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
  }, [setNodes])

  const handleTaskSizeChange = useCallback((taskId: string, width: number, height: number) => {
    console.log('Task size changed:', taskId, width, height)
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
  }, [setNodes])

  const handleTaskContextMenu = useCallback((e: React.MouseEvent, taskId: string) => {
    console.log('Task context menu:', taskId)
  }, [])

  const handleMemoryContentChange = useCallback((memoryId: string, content: string) => {
    console.log('Memory content changed:', memoryId, content)
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
  }, [setNodes])

  const handleMemoryContextMenu = useCallback((e: React.MouseEvent, memoryId: string) => {
    console.log('Memory context menu:', memoryId)
  }, [])

  const handleMemoryResize = useCallback((memoryId: string, width: number, height: number) => {
    console.log('Memory resized:', memoryId, width, height)
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
  }, [setNodes])

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
    (_event: React.MouseEvent, node: ZoneNode, _nodes: ZoneNode[]) => {
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
      }
    },
    [getPlansAtPosition, setNodes]
  )

  // --- Handle node changes ---
  const handleNodesChange: OnNodesChange<ZoneNode> = useCallback(
    (changes) => {
      onNodesChange(changes)

      // Log position changes
      for (const change of changes) {
        if (change.type === 'position' && 'position' in change && change.position && !change.dragging) {
          console.log('Node position finalized:', change.id, change.position)
        }
      }
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
    (connection) => {
      if (connection.source && connection.target) {
        console.log('New connection:', connection.source, '->', connection.target)
        
        setEdges((eds) =>
          addEdge(
            {
              ...connection,
              type: 'dependency',
              data: { relationshipType: 'DEPENDS_ON' }
            },
            eds
          )
        )
      }
    },
    [setEdges]
  )

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
        <div className="text-sm font-medium text-surface-800">{zone.name}</div>
        <div className="text-xs text-surface-500">
          {zone.plans.length} plans | {zone.plans.reduce((acc, p) => acc + p.tasks.length, 0)} tasks | {zone.memories.length} memories
        </div>
      </div>

      {/* Phase 0 Instructions */}
      <div className="absolute bottom-4 left-4 max-w-md bg-amber-50/95 backdrop-blur-sm px-4 py-3 rounded-lg shadow-sm border border-amber-200">
        <div className="text-sm font-semibold text-amber-800 mb-2">Phase 0 Prototype Testing</div>
        <ul className="text-xs text-amber-700 space-y-1">
          <li>1. Drag a <strong>Plan</strong> header - all tasks inside should move together</li>
          <li>2. Drag a <strong>Task</strong> to another plan - it will re-parent</li>
          <li>3. <strong>Resize</strong> plans/memories/tasks using corner handles when selected</li>
          <li>4. <strong>Edit</strong> plan description by double-clicking the left panel</li>
          <li>5. Connect tasks by dragging from one handle to another</li>
          <li>6. Positions and sizes now persist while you work</li>
        </ul>
      </div>
    </div>
  )
}
