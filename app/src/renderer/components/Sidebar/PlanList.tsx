import React from 'react'
import type { Plan, PlanStatus } from '../../types'

interface PlanListProps {
  plans: Plan[]
  selectedPlanId: string | null
  onSelectPlan: (planId: string) => void
  loading?: boolean
  error?: string | null
}

const statusColors: Record<PlanStatus, string> = {
  draft: 'bg-surface-200 text-surface-600',
  active: 'bg-primary-100 text-primary-700',
  completed: 'bg-green-100 text-green-700',
  archived: 'bg-surface-100 text-surface-500'
}

const statusLabels: Record<PlanStatus, string> = {
  draft: 'Draft',
  active: 'Active',
  completed: 'Done',
  archived: 'Archived'
}

export function PlanList({ plans, selectedPlanId, onSelectPlan, loading, error }: PlanListProps) {
  if (loading) {
    return (
      <div className="flex-1 flex items-center justify-center">
        <div className="text-surface-500 text-sm">Loading plans...</div>
      </div>
    )
  }
  
  if (error) {
    return (
      <div className="flex-1 p-4">
        <div className="text-red-600 text-sm">{error}</div>
      </div>
    )
  }
  
  if (plans.length === 0) {
    return (
      <div className="flex-1 flex items-center justify-center">
        <div className="text-surface-500 text-sm">No plans found</div>
      </div>
    )
  }
  
  return (
    <div className="flex-1 overflow-y-auto" role="listbox" aria-label="Plans">
      {plans.map((plan) => (
        <button
          key={plan.id}
          onClick={() => onSelectPlan(plan.id)}
          className={`w-full text-left p-3 border-b border-surface-100 hover:bg-surface-100 
                     transition-colors cursor-pointer ${
                       selectedPlanId === plan.id ? 'bg-primary-50 border-l-4 border-l-primary-500' : ''
                     }`}
          role="option"
          aria-selected={selectedPlanId === plan.id}
        >
          <div className="flex items-start justify-between gap-2">
            <div className="flex-1 min-w-0">
              <div className="font-medium text-sm truncate" title={plan.name}>
                {plan.name}
              </div>
              {plan.description && (
                <div className="text-xs text-surface-500 truncate mt-0.5" title={plan.description}>
                  {plan.description}
                </div>
              )}
            </div>
            <span className={`text-xs px-1.5 py-0.5 rounded ${statusColors[plan.status]}`}>
              {statusLabels[plan.status]}
            </span>
          </div>
          {plan.taskCount !== undefined && (
            <div className="text-xs text-surface-400 mt-1">
              {plan.taskCount} task{plan.taskCount !== 1 ? 's' : ''}
            </div>
          )}
        </button>
      ))}
    </div>
  )
}
