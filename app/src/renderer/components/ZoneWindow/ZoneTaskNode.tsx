import React, { useCallback, useState, memo } from 'react'
import { Handle, Position, type Node, NodeResizer } from '@xyflow/react'
import type { TaskInZone } from '../../types/zone'
import type { TaskStatus } from '../../types'

export interface ZoneTaskNodeData extends Record<string, unknown> {
  task: TaskInZone
  planId: string
  isSelected: boolean
  width: number
  height: number
  onContentChange?: (taskId: string, content: string) => void
  onSizeChange?: (taskId: string, width: number, height: number) => void
  onContextMenu?: (e: React.MouseEvent, taskId: string) => void
  onDrop?: (taskId: string, newPlanId: string) => void
}

export type ZoneTaskNodeType = Node<ZoneTaskNodeData, 'zoneTask'>

interface ZoneTaskNodeProps {
  data: ZoneTaskNodeData
  selected?: boolean
}

const MIN_WIDTH = 120
const MIN_HEIGHT = 80
export const TASK_DEFAULT_WIDTH = 200
export const TASK_DEFAULT_HEIGHT = 120

const statusColors: Record<TaskStatus, { bg: string; border: string; dot: string }> = {
  pending: { bg: 'bg-white', border: 'border-surface-300', dot: 'bg-surface-400' },
  in_progress: { bg: 'bg-blue-50', border: 'border-blue-300', dot: 'bg-blue-500' },
  completed: { bg: 'bg-green-50', border: 'border-green-300', dot: 'bg-green-500' },
  cancelled: { bg: 'bg-gray-100', border: 'border-gray-300', dot: 'bg-gray-400' },
  blocked: { bg: 'bg-red-50', border: 'border-red-300', dot: 'bg-red-500' }
}

function ZoneTaskNodeComponent({ data, selected }: ZoneTaskNodeProps) {
  const { task, isSelected, width, height, onContentChange, onSizeChange, onContextMenu } = data
  
  const showAsSelected = selected || isSelected
  
  const [isEditing, setIsEditing] = useState(false)
  const [content, setContent] = useState(task.content)

  const colors = statusColors[task.status] || statusColors.pending

  const handleContextMenu = useCallback((e: React.MouseEvent) => {
    e.preventDefault()
    e.stopPropagation()
    onContextMenu?.(e, task.id)
  }, [task.id, onContextMenu])

  const handleDoubleClick = useCallback((e: React.MouseEvent) => {
    e.stopPropagation()
    setIsEditing(true)
  }, [])

  const handleBlur = useCallback(() => {
    setIsEditing(false)
    if (content !== task.content) {
      onContentChange?.(task.id, content)
    }
  }, [task.id, task.content, content, onContentChange])

  const handleKeyDown = useCallback((e: React.KeyboardEvent) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault()
      handleBlur()
    } else if (e.key === 'Escape') {
      setContent(task.content)
      setIsEditing(false)
    }
  }, [task.content, handleBlur])

  const handleResizeEnd = useCallback((_event: unknown, params: { width: number; height: number }) => {
    onSizeChange?.(task.id, params.width, params.height)
  }, [task.id, onSizeChange])

  const shortId = task.id.substring(0, 6)

  return (
    <>
      {/* Resizer */}
      <NodeResizer
        isVisible={showAsSelected}
        minWidth={MIN_WIDTH}
        minHeight={MIN_HEIGHT}
        onResize={(_event, params) => {
          onSizeChange?.(task.id, params.width, params.height)
        }}
        lineClassName="!border-primary-400"
        handleClassName="!bg-primary-500 !border-primary-600"
      />

      {/* Connection handles */}
      <Handle
        type="target"
        position={Position.Left}
        id="target"
        className="!w-2.5 !h-2.5 !bg-primary-500 !border-2 !border-white"
      />
      <Handle
        type="source"
        position={Position.Right}
        id="source"
        className="!w-2.5 !h-2.5 !bg-primary-500 !border-2 !border-white"
      />
      
      <div
        className={`rounded-lg shadow-sm border-2 flex flex-col overflow-hidden
                   ${colors.bg} ${colors.border}
                   ${showAsSelected ? 'ring-2 ring-primary-400 shadow-md' : ''}`}
        style={{ width, height }}
        onContextMenu={handleContextMenu}
        onDoubleClick={handleDoubleClick}
      >
        {/* Header - draggable area */}
        <div className={`flex items-center justify-between px-2 py-1 border-b ${colors.border} cursor-move select-none`}>
          <div className="flex items-center gap-1.5 text-xs">
            <span className="font-mono text-surface-500">#{shortId}</span>
          </div>
          <div className="flex items-center gap-1.5">
            <span className="text-xs text-surface-500 capitalize">{task.status.replace('_', ' ')}</span>
            <div className={`w-2 h-2 rounded-full ${colors.dot}`} />
          </div>
        </div>
        
        {/* Content - editable, nodrag to prevent dragging when editing */}
        <div className="nodrag flex-1 overflow-hidden p-2">
          {isEditing ? (
            <textarea
              value={content}
              onChange={(e) => setContent(e.target.value)}
              onKeyDown={handleKeyDown}
              onBlur={handleBlur}
              autoFocus
              className="w-full h-full text-sm resize-none border-none outline-none bg-transparent"
              placeholder="Enter task description..."
            />
          ) : (
            <div className="text-sm text-surface-700 line-clamp-3">
              {task.content || <span className="italic text-surface-400">Double-click to edit...</span>}
            </div>
          )}
        </div>
      </div>
    </>
  )
}

export const ZoneTaskNode = memo(ZoneTaskNodeComponent)
