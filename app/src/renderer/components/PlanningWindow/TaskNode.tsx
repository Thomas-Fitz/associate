import React, { useCallback, useState, memo } from 'react'
import { Handle, Position, type Node } from '@xyflow/react'
import { TaskHeader } from './TaskHeader'
import { TaskDescription } from './TaskDescription'
import type { TaskInPlan } from '../../types'

export interface TaskNodeData extends Record<string, unknown> {
  task: TaskInPlan
  isSelected: boolean
  onContentChange: (taskId: string, content: string) => void
  onSizeChange: (taskId: string, width: number, height: number) => void
  onContextMenu: (e: React.MouseEvent, taskId: string) => void
}

export type TaskNodeType = Node<TaskNodeData, 'task'>

interface TaskNodeProps {
  data: TaskNodeData
}

const MIN_WIDTH = 150
const MIN_HEIGHT = 100
const DEFAULT_WIDTH = 250
const DEFAULT_HEIGHT = 150

function TaskNodeComponent({ data }: TaskNodeProps) {
  const { task, isSelected, onContentChange, onSizeChange, onContextMenu } = data
  
  const initialWidth = (task.metadata.ui_width as number) || DEFAULT_WIDTH
  const initialHeight = (task.metadata.ui_height as number) || DEFAULT_HEIGHT
  
  const [size, setSize] = useState({ width: initialWidth, height: initialHeight })
  
  const handleContentChange = useCallback((content: string) => {
    onContentChange(task.id, content)
  }, [task.id, onContentChange])
  
  const handleContextMenu = useCallback((e: React.MouseEvent) => {
    e.preventDefault()
    e.stopPropagation()
    onContextMenu(e, task.id)
  }, [task.id, onContextMenu])
  
  // Generic resize handler factory
  const createResizeHandler = useCallback((resizeType: 'corner' | 'right' | 'bottom') => {
    return (e: React.MouseEvent) => {
      e.preventDefault()
      e.stopPropagation()
      
      const startX = e.clientX
      const startY = e.clientY
      const startWidth = size.width
      const startHeight = size.height
      
      const handleMouseMove = (moveEvent: MouseEvent) => {
        moveEvent.preventDefault()
        moveEvent.stopPropagation()
        
        const deltaX = moveEvent.clientX - startX
        const deltaY = moveEvent.clientY - startY
        
        let newWidth = startWidth
        let newHeight = startHeight
        
        if (resizeType === 'corner' || resizeType === 'right') {
          newWidth = Math.max(MIN_WIDTH, startWidth + deltaX)
        }
        if (resizeType === 'corner' || resizeType === 'bottom') {
          newHeight = Math.max(MIN_HEIGHT, startHeight + deltaY)
        }
        
        setSize({ width: newWidth, height: newHeight })
      }
      
      const handleMouseUp = (upEvent: MouseEvent) => {
        upEvent.preventDefault()
        upEvent.stopPropagation()
        
        // Get final size from current state
        setSize(currentSize => {
          onSizeChange(task.id, currentSize.width, currentSize.height)
          return currentSize
        })
        
        document.removeEventListener('mousemove', handleMouseMove)
        document.removeEventListener('mouseup', handleMouseUp)
      }
      
      document.addEventListener('mousemove', handleMouseMove)
      document.addEventListener('mouseup', handleMouseUp)
    }
  }, [size.width, size.height, task.id, onSizeChange])
  
  // Calculate position number (1-based index)
  const positionNum = Math.floor(task.position / 1000)
  
  return (
    <>
      {/* Connection handles */}
      <Handle
        type="target"
        position={Position.Left}
        id="target"
        className="!w-3 !h-3 !bg-primary-500 !border-2 !border-white"
      />
      <Handle
        type="source"
        position={Position.Right}
        id="source"
        className="!w-3 !h-3 !bg-primary-500 !border-2 !border-white"
      />
      
      <div
        className={`bg-white rounded-lg shadow-md border-2 flex flex-col overflow-hidden relative
                   ${isSelected ? 'border-primary-500 shadow-lg' : 'border-surface-200'}`}
        style={{ width: size.width, height: size.height }}
        onContextMenu={handleContextMenu}
      >
        {/* Header - draggable area */}
        <TaskHeader
          taskId={task.id}
          position={positionNum}
          status={task.status}
          isSelected={isSelected}
        />
        
        {/* Description - editable, nodrag to prevent dragging when editing */}
        <div className="nodrag flex-1 overflow-hidden">
          <TaskDescription
            content={task.content}
            onChange={handleContentChange}
          />
        </div>
        
        {/* Resize handle - bottom right corner */}
        <div
          className="nodrag nopan absolute bottom-0 right-0 w-5 h-5 cursor-se-resize z-20 group"
          onMouseDown={createResizeHandler('corner')}
        >
          <svg 
            className="absolute bottom-0.5 right-0.5 w-3 h-3 text-surface-400 group-hover:text-primary-500"
            viewBox="0 0 10 10"
          >
            <path d="M 10 0 L 10 10 L 0 10" fill="none" stroke="currentColor" strokeWidth="2"/>
          </svg>
        </div>
        
        {/* Resize handle - right edge */}
        <div
          className="nodrag nopan absolute top-8 right-0 w-2 cursor-e-resize z-10 hover:bg-primary-300/30"
          style={{ height: 'calc(100% - 40px)' }}
          onMouseDown={createResizeHandler('right')}
        />
        
        {/* Resize handle - bottom edge */}
        <div
          className="nodrag nopan absolute bottom-0 left-0 h-2 cursor-s-resize z-10 hover:bg-primary-300/30"
          style={{ width: 'calc(100% - 20px)' }}
          onMouseDown={createResizeHandler('bottom')}
        />
      </div>
    </>
  )
}

export const TaskNode = memo(TaskNodeComponent)
