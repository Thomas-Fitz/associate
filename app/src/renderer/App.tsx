import React from 'react'
import { Sidebar } from './components/Sidebar'
import { ZoneWindow } from './components/ZoneWindow'
import { 
  CanvasContextMenu, 
  TaskContextMenu, 
  EdgeContextMenu, 
  ZoneContextMenu,
  PlanContextMenu,
  MemoryContextMenu 
} from './components/ContextMenu'
import { DeleteTaskDialog, DeleteEdgeDialog, DeleteZoneDialog } from './components/Dialogs'
import { useAppStore } from './stores/appStore'

export default function App() {
  const { contextMenu, hideContextMenu } = useAppStore()
  
  return (
    <div className="flex h-screen w-screen overflow-hidden">
      {/* Sidebar */}
      <Sidebar />
      
      {/* Main Zone Window */}
      <ZoneWindow />
      
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

      {contextMenu?.visible && contextMenu.type === 'zone' && contextMenu.zoneId && (
        <ZoneContextMenu
          x={contextMenu.x}
          y={contextMenu.y}
          zoneId={contextMenu.zoneId}
          onClose={hideContextMenu}
        />
      )}

      {contextMenu?.visible && contextMenu.type === 'plan' && contextMenu.planId && (
        <PlanContextMenu
          x={contextMenu.x}
          y={contextMenu.y}
          planId={contextMenu.planId}
          onClose={hideContextMenu}
        />
      )}

      {contextMenu?.visible && contextMenu.type === 'memory' && contextMenu.memoryId && (
        <MemoryContextMenu
          x={contextMenu.x}
          y={contextMenu.y}
          memoryId={contextMenu.memoryId}
          onClose={hideContextMenu}
        />
      )}
      
      {/* Dialogs */}
      <DeleteTaskDialog />
      <DeleteEdgeDialog />
      <DeleteZoneDialog />
    </div>
  )
}
