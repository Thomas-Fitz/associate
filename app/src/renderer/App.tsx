import React, { useState } from 'react'
import { Sidebar } from './components/Sidebar'
import { PlanningWindow } from './components/PlanningWindow'
import { ZoneWindow } from './components/ZoneWindow'
import { CanvasContextMenu, TaskContextMenu, EdgeContextMenu } from './components/ContextMenu'
import { DeleteTaskDialog, DeleteEdgeDialog } from './components/Dialogs'
import { useAppStore } from './stores/appStore'

export default function App() {
  const { contextMenu, hideContextMenu } = useAppStore()
  
  // Phase 0 Prototype: Toggle between PlanningWindow and ZoneWindow
  const [useZonePrototype, setUseZonePrototype] = useState(true)
  
  return (
    <div className="flex h-screen w-screen overflow-hidden">
      {/* Phase 0 Toggle Button */}
      <div className="absolute top-2 right-2 z-50">
        <button
          onClick={() => setUseZonePrototype(!useZonePrototype)}
          className={`px-3 py-1.5 text-xs font-medium rounded-md shadow-sm border transition-colors
                     ${useZonePrototype 
                       ? 'bg-amber-500 text-white border-amber-600 hover:bg-amber-600' 
                       : 'bg-white text-surface-700 border-surface-300 hover:bg-surface-50'}`}
        >
          {useZonePrototype ? 'Zone Prototype (Phase 0)' : 'Original View'}
        </button>
      </div>

      {/* Sidebar - only show for original view */}
      {!useZonePrototype && <Sidebar />}
      
      {/* Main Window - toggle between original and prototype */}
      {useZonePrototype ? <ZoneWindow /> : <PlanningWindow />}
      
      {/* Context Menus */}
      {contextMenu?.visible && contextMenu.type === 'canvas' && (
        <CanvasContextMenu
          x={contextMenu.x}
          y={contextMenu.y}
          canvasX={contextMenu.canvasX}
          canvasY={contextMenu.canvasY}
          onClose={hideContextMenu}
        />
      )}
      
      {contextMenu?.visible && contextMenu.type === 'task' && contextMenu.taskId && (
        <TaskContextMenu
          x={contextMenu.x}
          y={contextMenu.y}
          taskId={contextMenu.taskId}
          onClose={hideContextMenu}
        />
      )}
      
      {contextMenu?.visible && contextMenu.type === 'edge' && contextMenu.edgeId && (
        <EdgeContextMenu
          x={contextMenu.x}
          y={contextMenu.y}
          edgeId={contextMenu.edgeId}
          onClose={hideContextMenu}
        />
      )}
      
      {/* Dialogs */}
      <DeleteTaskDialog />
      <DeleteEdgeDialog />
    </div>
  )
}
