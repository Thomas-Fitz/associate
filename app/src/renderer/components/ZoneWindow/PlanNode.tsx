import React, { memo, useCallback, useState } from 'react'
import { Handle, Position, type Node, NodeResizer } from '@xyflow/react'
import type { PlanInZone } from '../../types/zone'

export interface PlanNodeData extends Record<string, unknown> {
  plan: PlanInZone
  isSelected: boolean
  width: number
  height: number
  onContextMenu?: (e: React.MouseEvent, planId: string) => void
  onResize?: (planId: string, width: number, height: number) => void
  onDescriptionChange?: (planId: string, description: string) => void
}

export type PlanNodeType = Node<PlanNodeData, 'plan'>

interface PlanNodeProps {
  data: PlanNodeData
  selected?: boolean
}

// Default dimensions for plan groups
export const PLAN_DEFAULT_WIDTH = 500
export const PLAN_DEFAULT_HEIGHT = 350
export const PLAN_MIN_WIDTH = 400
export const PLAN_MIN_HEIGHT = 250
const DESCRIPTION_WIDTH = 150

function PlanNodeComponent({ data, selected }: PlanNodeProps) {
  const { plan, isSelected, width, height, onContextMenu, onResize, onDescriptionChange } = data
  const showAsSelected = selected || isSelected

  const [isEditingDescription, setIsEditingDescription] = useState(false)
  const [descriptionValue, setDescriptionValue] = useState(plan.description)

  const handleContextMenu = useCallback((e: React.MouseEvent) => {
    e.preventDefault()
    e.stopPropagation()
    onContextMenu?.(e, plan.id)
  }, [plan.id, onContextMenu])

  const handleDescriptionDoubleClick = useCallback(() => {
    setIsEditingDescription(true)
  }, [])

  const handleDescriptionBlur = useCallback(() => {
    setIsEditingDescription(false)
    if (descriptionValue !== plan.description) {
      onDescriptionChange?.(plan.id, descriptionValue)
    }
  }, [plan.id, plan.description, descriptionValue, onDescriptionChange])

  const handleDescriptionKeyDown = useCallback((e: React.KeyboardEvent) => {
    if (e.key === 'Escape') {
      setDescriptionValue(plan.description)
      setIsEditingDescription(false)
    }
  }, [plan.description])

  // Status color mapping
  const statusColors: Record<string, { bg: string; text: string; border: string; descBg: string }> = {
    draft: { bg: 'bg-gray-100', text: 'text-gray-700', border: 'border-gray-300', descBg: 'bg-gray-200/50' },
    active: { bg: 'bg-blue-50', text: 'text-blue-700', border: 'border-blue-300', descBg: 'bg-blue-100/50' },
    completed: { bg: 'bg-green-50', text: 'text-green-700', border: 'border-green-300', descBg: 'bg-green-100/50' },
    archived: { bg: 'bg-yellow-50', text: 'text-yellow-700', border: 'border-yellow-300', descBg: 'bg-yellow-100/50' }
  }

  const colors = statusColors[plan.status] || statusColors.draft

  return (
    <>
      {/* Resizer - only visible when selected */}
      <NodeResizer
        isVisible={showAsSelected}
        minWidth={PLAN_MIN_WIDTH}
        minHeight={PLAN_MIN_HEIGHT}
        onResize={(_event, params) => {
          // Called during resize - we update via onResize callback
          onResize?.(plan.id, params.width, params.height)
        }}
        lineClassName="!border-primary-400"
        handleClassName="!bg-primary-500 !border-primary-600"
      />

      {/* Plan group container - use width/height from data props */}
      <div
        className={`rounded-xl border-2 ${colors.border} ${colors.bg} ${showAsSelected ? 'ring-2 ring-primary-400' : ''} flex flex-col`}
        style={{ 
          width, 
          height,
          minWidth: PLAN_MIN_WIDTH,
          minHeight: PLAN_MIN_HEIGHT
        }}
        onContextMenu={handleContextMenu}
      >
        {/* Header - draggable area */}
        <div className={`px-4 py-2 ${colors.text} border-b ${colors.border} rounded-t-xl cursor-move flex items-center justify-between shrink-0`}>
          <div className="flex items-center gap-2">
            <span className="font-semibold text-sm truncate">{plan.name}</span>
            <span className="text-xs px-2 py-0.5 rounded-full bg-white/50 capitalize">
              {plan.status}
            </span>
          </div>
          <span className="text-xs text-opacity-70">
            {plan.tasks.length} task{plan.tasks.length !== 1 ? 's' : ''}
          </span>
        </div>

        {/* Main content area - flex row with description on left */}
        <div className="flex flex-1 min-h-0">
          {/* Description panel on the left */}
          <div 
            className={`${colors.descBg} border-r ${colors.border} p-2 nodrag flex flex-col shrink-0`}
            style={{ width: DESCRIPTION_WIDTH }}
          >
            <div className="text-xs font-medium text-gray-500 mb-1">Description</div>
            {isEditingDescription ? (
              <textarea
                value={descriptionValue}
                onChange={(e) => setDescriptionValue(e.target.value)}
                onBlur={handleDescriptionBlur}
                onKeyDown={handleDescriptionKeyDown}
                autoFocus
                className="flex-1 text-xs resize-none border-none outline-none bg-white/50 rounded p-1"
                placeholder="Add plan description..."
              />
            ) : (
              <div 
                className="flex-1 text-xs text-gray-600 overflow-auto cursor-text hover:bg-white/30 rounded p-1"
                onDoubleClick={handleDescriptionDoubleClick}
              >
                {plan.description || <span className="italic text-gray-400">Double-click to add...</span>}
              </div>
            )}
          </div>

          {/* Task area - where child nodes go */}
          <div className="flex-1 p-2 nodrag relative">
            {/* Child task nodes will be positioned inside this area via ReactFlow's parentId */}
            {plan.tasks.length === 0 && (
              <div className="absolute inset-0 flex items-center justify-center text-sm text-gray-400 italic pointer-events-none">
                Drop tasks here
              </div>
            )}
          </div>
        </div>
      </div>

      {/* Connection handles for plan-to-plan or memory-to-plan edges */}
      <Handle
        type="target"
        position={Position.Left}
        id="target"
        className="!w-2 !h-4 !rounded-sm !bg-primary-400 !border-primary-600"
      />
      <Handle
        type="source"
        position={Position.Right}
        id="source"
        className="!w-2 !h-4 !rounded-sm !bg-primary-400 !border-primary-600"
      />
    </>
  )
}

export const PlanNode = memo(PlanNodeComponent)
