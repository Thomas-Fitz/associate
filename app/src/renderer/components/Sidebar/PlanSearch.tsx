import React, { useState, useEffect } from 'react'
import { useAppStore } from '../../stores/appStore'
import type { PlanStatus } from '../../types'

export function PlanSearch() {
  const { searchQuery, statusFilter, setSearchQuery, setStatusFilter } = useAppStore()
  const [localSearch, setLocalSearch] = useState(searchQuery)
  
  // Debounce search input
  useEffect(() => {
    const timer = setTimeout(() => {
      setSearchQuery(localSearch)
    }, 300)
    
    return () => clearTimeout(timer)
  }, [localSearch, setSearchQuery])
  
  return (
    <div className="p-3 border-b border-surface-200 space-y-2">
      {/* Search input */}
      <input
        type="text"
        placeholder="Search plans..."
        value={localSearch}
        onChange={(e) => setLocalSearch(e.target.value)}
        className="w-full px-3 py-2 text-sm border border-surface-300 rounded-md 
                   focus:outline-none focus:ring-2 focus:ring-primary-500 focus:border-transparent
                   placeholder:text-surface-400"
        aria-label="Search plans"
      />
      
      {/* Status filter */}
      <select
        value={statusFilter}
        onChange={(e) => setStatusFilter(e.target.value as PlanStatus | 'all')}
        className="w-full px-3 py-2 text-sm border border-surface-300 rounded-md 
                   focus:outline-none focus:ring-2 focus:ring-primary-500 focus:border-transparent
                   bg-white"
        aria-label="Filter by status"
      >
        <option value="all">All Status</option>
        <option value="draft">Draft</option>
        <option value="active">Active</option>
        <option value="completed">Completed</option>
        <option value="archived">Archived</option>
      </select>
    </div>
  )
}
