import React, { useCallback, useState, memo } from 'react'
import { Handle, Position, type Node } from '@xyflow/react'
import { Resizable } from 'react-resizable'
import { TaskHeader } from './TaskHeader'
import { TaskDescription } from './TaskDescription'
import type { TaskInPlan } from '../../types'
import 'react-resizable/css/styles.css'

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
  
  const initialWidth = task.metadata.ui_width || DEFAULT_WIDTH
  const initialHeight = task.metadata.ui_height || DEFAULT_HEIGHT
  
  const [size, setSize] = useState({ width: initialWidth, height: initialHeight })
  
  const handleResize = useCallback((_e: React.SyntheticEvent, { size: newSize }: { size: { width: number; height: number } }) => {
    setSize(newSize)
  }, [])
  
  const handleResizeStop = useCallback((_e: React.SyntheticEvent, { size: newSize }: { size: { width: number; height: number } }) => {
    setSize(newSize)
    onSizeChange(task.id, newSize.width, newSize.height)
  }, [task.id, onSizeChange])
  
  const handleContentChange = useCallback((content: string) => {
    onContentChange(task.id, content)
  }, [task.id, onContentChange])
  
  const handleContextMenu = useCallback((e: React.MouseEvent) => {
    e.preventDefault()
    e.stopPropagation()
    onContextMenu(e, task.id)
  }, [task.id, onContextMenu])
  
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
      
      <Resizable
        width={size.width}
        height={size.height}
        minConstraints={[MIN_WIDTH, MIN_HEIGHT]}
        onResize={handleResize}
        onResizeStop={handleResizeStop}
        resizeHandles={['se', 'e', 's']}
      >
        <div
          className={`bg-white rounded-lg shadow-md border-2 flex flex-col overflow-hidden
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
          
          {/* Description - editable */}
          <TaskDescription
            content={task.content}
            onChange={handleContentChange}
          />
        </div>
      </Resizable>
    </>
  )
}

export const TaskNode = memo(TaskNodeComponent)
