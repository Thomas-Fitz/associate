import { useCallback, useEffect } from 'react'
import { useAppStore } from '../stores/appStore'
import { useDatabase } from './useDatabase'
import type { PlanStatus } from '../types'

export function usePlans() {
  const db = useDatabase()
  const {
    plans,
    selectedPlanId,
    selectedPlan,
    plansLoading,
    plansError,
    searchQuery,
    statusFilter,
    setPlans,
    setSelectedPlanId,
    setSelectedPlan,
    setPlansLoading,
    setPlansError
  } = useAppStore()
  
  // Load plans list
  const loadPlans = useCallback(async () => {
    setPlansLoading(true)
    setPlansError(null)
    
    try {
      const options: { status?: PlanStatus; search?: string } = {}
      
      if (statusFilter !== 'all') {
        options.status = statusFilter
      }
      if (searchQuery.trim()) {
        options.search = searchQuery.trim()
      }
      
      const loadedPlans = await db.plans.list(options)
      setPlans(loadedPlans)
    } catch (err) {
      console.error('Failed to load plans:', err)
      setPlansError(err instanceof Error ? err.message : 'Failed to load plans')
    } finally {
      setPlansLoading(false)
    }
  }, [db.plans, searchQuery, statusFilter, setPlans, setPlansLoading, setPlansError])
  
  // Load selected plan with tasks
  const loadSelectedPlan = useCallback(async (planId: string) => {
    try {
      const plan = await db.plans.get(planId)
      setSelectedPlan(plan)
    } catch (err) {
      console.error('Failed to load plan:', err)
      setSelectedPlan(null)
    }
  }, [db.plans, setSelectedPlan])
  
  // Select a plan
  const selectPlan = useCallback((planId: string | null) => {
    setSelectedPlanId(planId)
    if (planId) {
      loadSelectedPlan(planId)
    } else {
      setSelectedPlan(null)
    }
  }, [setSelectedPlanId, setSelectedPlan, loadSelectedPlan])
  
  // Load plans on mount and when filters change
  useEffect(() => {
    loadPlans()
  }, [loadPlans])
  
  return {
    plans,
    selectedPlanId,
    selectedPlan,
    loading: plansLoading,
    error: plansError,
    selectPlan,
    refreshPlans: loadPlans,
    refreshSelectedPlan: () => selectedPlanId && loadSelectedPlan(selectedPlanId)
  }
}
