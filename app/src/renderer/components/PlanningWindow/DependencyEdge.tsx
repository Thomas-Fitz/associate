import React from 'react'
import { BaseEdge, getBezierPath, type Edge, Position } from '@xyflow/react'

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
  // Calculate curvature based on distance - limit the curve for long edges
  const distance = Math.sqrt(Math.pow(targetX - sourceX, 2) + Math.pow(targetY - sourceY, 2))
  const curvature = Math.min(0.25, 50 / distance) // Reduce curvature for longer edges
  
  const [edgePath] = getBezierPath({
    sourceX,
    sourceY,
    sourcePosition,
    targetX,
    targetY,
    targetPosition,
    curvature
  })
  
  const isBlocks = data?.relationshipType === 'BLOCKS'
  
  return (
    <BaseEdge
      id={id}
      path={edgePath}
      style={{
        stroke: isBlocks ? '#f97316' : '#64748b',
        strokeWidth: selected ? 3 : 2,
        strokeDasharray: isBlocks ? '5 5' : undefined,
      }}
      markerEnd={`url(#dependency-arrow${isBlocks ? '-blocks' : ''})`}
    />
  )
}

// SVG marker definitions for arrows
export function DependencyArrowMarker() {
  return (
    <svg style={{ position: 'absolute', width: 0, height: 0 }}>
      <defs>
        <marker
          id="dependency-arrow"
          viewBox="0 0 10 10"
          refX="10"
          refY="5"
          markerWidth="6"
          markerHeight="6"
          orient="auto-start-reverse"
        >
          <path d="M 0 0 L 10 5 L 0 10 z" fill="#64748b" />
        </marker>
        <marker
          id="dependency-arrow-blocks"
          viewBox="0 0 10 10"
          refX="10"
          refY="5"
          markerWidth="6"
          markerHeight="6"
          orient="auto-start-reverse"
        >
          <path d="M 0 0 L 10 5 L 0 10 z" fill="#f97316" />
        </marker>
      </defs>
    </svg>
  )
}
