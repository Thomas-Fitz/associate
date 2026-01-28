import React from 'react'
import { ContextMenu, ContextMenuItem, ContextMenuSeparator } from './ContextMenu'
import { useZones } from '../../hooks/useZones'
import { useAppStore } from '../../stores/appStore'

interface TerminalContextMenuProps {
  x: number
  y: number
  terminalId: string
  onClose: () => void
}

/**
 * Context menu for terminal nodes on the zone canvas.
 * Provides actions like Copy, Paste, Clear, Kill, Rename, Restart, and Close.
 */
export function TerminalContextMenu({ x, y, terminalId, onClose }: TerminalContextMenuProps) {
  const { selectedZone } = useZones()
  const { showDeleteTerminalDialog } = useAppStore()
  
  const terminal = selectedZone?.terminals?.find(t => t.id === terminalId)
  const isRunning = terminal?.state.status === 'running'
  
  const handleCopy = () => {
    // Trigger copy via custom event - the TerminalNode will handle it
    const event = new CustomEvent('terminal:copy', { detail: { terminalId } })
    window.dispatchEvent(event)
    onClose()
  }
  
  const handlePaste = () => {
    // Trigger paste via custom event - the TerminalNode will handle it
    const event = new CustomEvent('terminal:paste', { detail: { terminalId } })
    window.dispatchEvent(event)
    onClose()
  }
  
  const handleClearScreen = () => {
    // Send clear screen sequence to the terminal
    const event = new CustomEvent('terminal:clear', { detail: { terminalId } })
    window.dispatchEvent(event)
    onClose()
  }
  
  const handleKillProcess = async () => {
    if (!isRunning) {
      onClose()
      return
    }
    
    try {
      await window.electronAPI?.pty.kill(terminalId)
    } catch (err) {
      console.error('Failed to kill terminal process:', err)
    }
    onClose()
  }
  
  const handleRename = () => {
    // Trigger inline rename via custom event
    const event = new CustomEvent('terminal:rename', { detail: { terminalId } })
    window.dispatchEvent(event)
    onClose()
  }
  
  const handleDuplicate = () => {
    // Trigger duplicate via custom event - will create a new terminal with same config
    const event = new CustomEvent('terminal:duplicate', { detail: { terminalId } })
    window.dispatchEvent(event)
    onClose()
  }
  
  const handleRestart = async () => {
    // Kill existing process if running, then start a new one
    try {
      if (isRunning) {
        await window.electronAPI?.pty.kill(terminalId)
      }
      
      // Small delay to allow cleanup
      await new Promise(resolve => setTimeout(resolve, 100))
      
      // Start new PTY with same config
      if (terminal?.config) {
        await window.electronAPI?.pty.create(terminalId, terminal.config)
      }
    } catch (err) {
      console.error('Failed to restart terminal:', err)
    }
    onClose()
  }
  
  const handleConnect = async () => {
    // Connect/reconnect the terminal PTY
    try {
      if (terminal?.config) {
        await window.electronAPI?.pty.create(terminalId, terminal.config)
      }
    } catch (err) {
      console.error('Failed to connect terminal:', err)
    }
    onClose()
  }
  
  const handleClose = () => {
    if (!terminal) {
      onClose()
      return
    }
    
    showDeleteTerminalDialog(terminalId, terminal.name)
    onClose()
  }
  
  return (
    <ContextMenu x={x} y={y} onClose={onClose}>
      <ContextMenuItem onClick={handleCopy}>
        Copy
        <span className="ml-auto text-xs text-surface-400">Ctrl+Shift+C</span>
      </ContextMenuItem>
      <ContextMenuItem onClick={handlePaste}>
        Paste
        <span className="ml-auto text-xs text-surface-400">Ctrl+Shift+V</span>
      </ContextMenuItem>
      
      <ContextMenuSeparator />
      
      <ContextMenuItem onClick={handleClearScreen} disabled={!isRunning}>
        Clear Screen
      </ContextMenuItem>
      <ContextMenuItem onClick={handleKillProcess} disabled={!isRunning} danger>
        Kill Process
      </ContextMenuItem>
      
      <ContextMenuSeparator />
      
      <ContextMenuItem onClick={handleRename}>
        Rename
      </ContextMenuItem>
      <ContextMenuItem onClick={handleDuplicate}>
        Duplicate
      </ContextMenuItem>
      
      <ContextMenuSeparator />
      
      {!isRunning && terminal?.state.status !== 'running' && (
        <ContextMenuItem onClick={handleConnect}>
          Connect
        </ContextMenuItem>
      )}
      <ContextMenuItem onClick={handleRestart}>
        Restart
      </ContextMenuItem>
      
      <ContextMenuSeparator />
      
      <ContextMenuItem onClick={handleClose} danger>
        Close Terminal
      </ContextMenuItem>
    </ContextMenu>
  )
}
