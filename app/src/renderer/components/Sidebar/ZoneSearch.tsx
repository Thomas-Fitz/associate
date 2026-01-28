import React, { useState, useEffect } from 'react'
import { useAppStore } from '../../stores/appStore'

export function ZoneSearch() {
  const { searchQuery, setSearchQuery } = useAppStore()
  const [localSearch, setLocalSearch] = useState(searchQuery)
  
  // Debounce search input
  useEffect(() => {
    const timer = setTimeout(() => {
      setSearchQuery(localSearch)
    }, 300)
    
    return () => clearTimeout(timer)
  }, [localSearch, setSearchQuery])
  
  return (
    <div className="p-3 border-b border-surface-200">
      {/* Search input */}
      <input
        type="text"
        placeholder="Search zones..."
        value={localSearch}
        onChange={(e) => setLocalSearch(e.target.value)}
        className="w-full px-3 py-2 text-sm border border-surface-300 rounded-md 
                   focus:outline-none focus:ring-2 focus:ring-primary-500 focus:border-transparent
                   placeholder:text-surface-400"
        aria-label="Search zones"
      />
      {/* No status filter - removed per Zone refactor requirements */}
    </div>
  )
}
