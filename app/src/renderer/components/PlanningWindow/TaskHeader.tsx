import React from 'react'
import type { TaskStatus } from '../../types'

interface TaskHeaderProps {
  taskId: string
  position: number
  status: TaskStatus
  isSelected?: boolean
}

const statusColors: Record<TaskStatus, string> = {
  pending: 'bg-surface-200',
  in_progress: 'bg-primary-500',
  completed: 'bg-green-500',
  cancelled: 'bg-surface-400',
  blocked: 'bg-red-500'
}

export function TaskHeader({ taskId, position, status, isSelected }: TaskHeaderProps) {
  const shortId = taskId.substring(0, 8)
  
  return (
    <div
      className={`flex items-center justify-between px-2 py-1.5 border-b
                  cursor-move select-none
                  ${isSelected ? 'bg-primary-100 border-primary-200' : 'bg-surface-100 border-surface-200'}`}
    >
      <div className="flex items-center gap-2 text-xs">
        <span className="font-mono text-surface-600">#{position}</span>
        <span className="text-surface-400">|</span>
        <span className="font-mono text-surface-500" title={taskId}>{shortId}</span>
      </div>
      <div
        className={`w-2 h-2 rounded-full ${statusColors[status]}`}
        title={status.replace('_', ' ')}
      />
    </div>
  )
}
