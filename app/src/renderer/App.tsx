import React from 'react'
import { Sidebar } from './components/Sidebar'
import { PlanningWindow } from './components/PlanningWindow'
import { CanvasContextMenu, TaskContextMenu } from './components/ContextMenu'
import { DeleteTaskDialog } from './components/Dialogs'
import { useAppStore } from './stores/appStore'

export default function App() {
  const { contextMenu, hideContextMenu } = useAppStore()
  
  return (
    <div className="flex h-screen w-screen overflow-hidden">
      {/* Sidebar */}
      <Sidebar />
      
      {/* Main Planning Window */}
      <PlanningWindow />
      
      {/* Context Menus */}
      {contextMenu?.visible && contextMenu.type === 'canvas' && (
        <CanvasContextMenu
          x={contextMenu.x}
          y={contextMenu.y}
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
      
      {/* Dialogs */}
      <DeleteTaskDialog />
    </div>
  )
}
