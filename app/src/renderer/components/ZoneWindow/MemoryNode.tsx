import React, { memo, useCallback, useState } from 'react'
import { Handle, Position, type Node, NodeResizer } from '@xyflow/react'
import type { MemoryInZone } from '../../types/zone'

export interface MemoryNodeData extends Record<string, unknown> {
  memory: MemoryInZone
  isSelected: boolean
  width: number
  height: number
  onContentChange?: (memoryId: string, content: string) => void
  onContextMenu?: (e: React.MouseEvent, memoryId: string) => void
  onResize?: (memoryId: string, width: number, height: number) => void
}

export type MemoryNodeType = Node<MemoryNodeData, 'memory'>

interface MemoryNodeProps {
  data: MemoryNodeData
  selected?: boolean
}

export const MEMORY_DEFAULT_WIDTH = 200
export const MEMORY_DEFAULT_HEIGHT = 120
const MIN_WIDTH = 150
const MIN_HEIGHT = 80

function MemoryNodeComponent({ data, selected }: MemoryNodeProps) {
  const { memory, isSelected, width, height, onContentChange, onContextMenu, onResize } = data
  const showAsSelected = selected || isSelected

  const [isEditing, setIsEditing] = useState(false)
  const [content, setContent] = useState(memory.content)

  const handleContextMenu = useCallback((e: React.MouseEvent) => {
    e.preventDefault()
    e.stopPropagation()
    onContextMenu?.(e, memory.id)
  }, [memory.id, onContextMenu])

  const handleDoubleClick = useCallback(() => {
    setIsEditing(true)
  }, [])

  const handleBlur = useCallback(() => {
    setIsEditing(false)
    if (content !== memory.content) {
      onContentChange?.(memory.id, content)
    }
  }, [memory.id, memory.content, content, onContentChange])

  const handleResizeEnd = useCallback((_event: unknown, params: { width: number; height: number }) => {
    onResize?.(memory.id, params.width, params.height)
  }, [memory.id, onResize])

  // Type color mapping
  const typeColors: Record<string, { bg: string; border: string; icon: string }> = {
    Note: { bg: 'bg-amber-50', border: 'border-amber-300', icon: '📝' },
    Repository: { bg: 'bg-purple-50', border: 'border-purple-300', icon: '📦' },
    Memory: { bg: 'bg-cyan-50', border: 'border-cyan-300', icon: '💾' }
  }

  const colors = typeColors[memory.type] || typeColors.Memory

  return (
    <>
      {/* Resizer */}
      <NodeResizer
        isVisible={showAsSelected}
        minWidth={MIN_WIDTH}
        minHeight={MIN_HEIGHT}
        onResize={(_event, params) => {
          onResize?.(memory.id, params.width, params.height)
        }}
        lineClassName="!border-amber-400"
        handleClassName="!bg-amber-500 !border-amber-600"
      />

      {/* Memory container */}
      <div
        className={`rounded-lg border-2 ${colors.border} ${colors.bg} shadow-sm 
                   ${showAsSelected ? 'ring-2 ring-amber-400 shadow-md' : ''}`}
        style={{ width, height }}
        onContextMenu={handleContextMenu}
        onDoubleClick={handleDoubleClick}
      >
        {/* Header */}
        <div className={`px-3 py-1.5 border-b ${colors.border} flex items-center gap-2 cursor-move`}>
          <span className="text-sm">{colors.icon}</span>
          <span className="text-xs font-medium text-gray-600 uppercase tracking-wide">
            {memory.type}
          </span>
        </div>

        {/* Content area */}
        <div className="p-2 h-full nodrag overflow-hidden">
          {isEditing ? (
            <textarea
              value={content}
              onChange={(e) => setContent(e.target.value)}
              onBlur={handleBlur}
              autoFocus
              className="w-full h-full text-sm resize-none border-none outline-none bg-transparent"
              placeholder="Enter memory content..."
            />
          ) : (
            <div className="text-sm text-gray-700 line-clamp-4 whitespace-pre-wrap">
              {memory.content || <span className="italic text-gray-400">Double-click to edit...</span>}
            </div>
          )}
        </div>
      </div>

      {/* Connection handles */}
      <Handle
        type="target"
        position={Position.Left}
        id="target"
        className="!w-2 !h-2 !bg-amber-500 !border-amber-700"
      />
      <Handle
        type="source"
        position={Position.Right}
        id="source"
        className="!w-2 !h-2 !bg-amber-500 !border-amber-700"
      />
    </>
  )
}

export const MemoryNode = memo(MemoryNodeComponent)
