import React from 'react'
import { PlanSearch } from './PlanSearch'
import { PlanList } from './PlanList'
import { usePlans } from '../../hooks'

export function Sidebar() {
  const { plans, selectedPlanId, selectPlan, loading, error } = usePlans()
  
  return (
    <aside 
      className="w-sidebar flex flex-col bg-white border-r border-surface-200 h-full"
      aria-label="Sidebar"
    >
      {/* Header */}
      <div className="p-3 border-b border-surface-200">
        <h1 className="text-lg font-semibold text-surface-800">Associate Planner</h1>
      </div>
      
      {/* Search and Filter */}
      <PlanSearch />
      
      {/* Plan List */}
      <PlanList
        plans={plans}
        selectedPlanId={selectedPlanId}
        onSelectPlan={selectPlan}
        loading={loading}
        error={error}
      />
    </aside>
  )
}
