import React from 'react'
import { BaseEdge, getBezierPath, EdgeLabelRenderer, type Edge, Position } from '@xyflow/react'

export interface DependencyEdgeData extends Record<string, unknown> {
  relationshipType: 'DEPENDS_ON' | 'BLOCKS'
}

export type DependencyEdgeType = Edge<DependencyEdgeData, 'dependency'>

interface DependencyEdgeProps {
  id: string
  sourceX: number
  sourceY: number
  targetX: number
  targetY: number
  sourcePosition: Position
  targetPosition: Position
  data?: DependencyEdgeData
  selected?: boolean
}

export function DependencyEdge({
  id,
  sourceX,
  sourceY,
  targetX,
  targetY,
  sourcePosition,
  targetPosition,
  data,
  selected
}: DependencyEdgeProps) {
  const [edgePath, labelX, labelY] = getBezierPath({
    sourceX,
    sourceY,
    sourcePosition,
    targetX,
    targetY,
    targetPosition
  })
  
  const isBlocks = data?.relationshipType === 'BLOCKS'
  
  return (
    <>
      <BaseEdge
        id={id}
        path={edgePath}
        style={{
          stroke: isBlocks ? '#f97316' : '#a1a1aa', // orange for BLOCKS, gray for DEPENDS_ON
          strokeWidth: selected ? 3 : 2,
          strokeDasharray: isBlocks ? '5 5' : undefined
        }}
        markerEnd="url(#dependency-arrow)"
      />
      
      {/* Edge label (optional, shown on hover or selection) */}
      {selected && (
        <EdgeLabelRenderer>
          <div
            style={{
              position: 'absolute',
              transform: `translate(-50%, -50%) translate(${labelX}px,${labelY}px)`,
              pointerEvents: 'all'
            }}
            className="px-2 py-1 text-xs bg-white border border-surface-200 rounded shadow-sm"
          >
            {isBlocks ? 'blocks' : 'depends on'}
          </div>
        </EdgeLabelRenderer>
      )}
    </>
  )
}

// SVG marker definition for arrows
export function DependencyArrowMarker() {
  return (
    <svg style={{ position: 'absolute', width: 0, height: 0 }}>
      <defs>
        <marker
          id="dependency-arrow"
          viewBox="0 0 10 10"
          refX="8"
          refY="5"
          markerWidth="6"
          markerHeight="6"
          orient="auto-start-reverse"
        >
          <path d="M 0 0 L 10 5 L 0 10 z" fill="#a1a1aa" />
        </marker>
      </defs>
    </svg>
  )
}
