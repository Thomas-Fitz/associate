import React, { memo, useCallback, useState, useRef, useEffect } from 'react'
import { type Node, NodeResizer } from '@xyflow/react'
import { TerminalErrorBoundary } from './TerminalErrorBoundary'
import { useTerminal } from '../../hooks/useTerminal'
import type { TerminalInZone, TerminalState } from '../../types/terminal'
import 'xterm/css/xterm.css'

export interface TerminalNodeData extends Record<string, unknown> {
  terminal: TerminalInZone
  isSelected: boolean
  width: number
  height: number
  onNameChange?: (terminalId: string, name: string) => void
  onResize?: (terminalId: string, width: number, height: number) => void
  onContextMenu?: (e: React.MouseEvent, terminalId: string) => void
  onFocus?: (terminalId: string) => void
  onStateChange?: (terminalId: string, state: TerminalState) => void
}

export type TerminalNodeType = Node<TerminalNodeData, 'terminal'>

interface TerminalNodeProps {
  data: TerminalNodeData
  selected?: boolean
}

// Terminal size constants
export const TERMINAL_DEFAULT_WIDTH = 600
export const TERMINAL_DEFAULT_HEIGHT = 400
const MIN_WIDTH = 400
const MIN_HEIGHT = 300
const COLLAPSED_SIZE = 60
const HEADER_HEIGHT = 36

// Status colors
const STATUS_COLORS: Record<TerminalState['status'], { border: string; indicator: string; bg: string }> = {
  running: { border: 'border-green-500', indicator: 'bg-green-500', bg: 'bg-green-500/10' },
  stopped: { border: 'border-gray-500', indicator: 'bg-gray-500', bg: 'bg-gray-500/10' },
  error: { border: 'border-red-500', indicator: 'bg-red-500', bg: 'bg-red-500/10' },
  disconnected: { border: 'border-gray-600', indicator: 'bg-gray-600', bg: 'bg-gray-600/10' }
}

function TerminalNodeInner({ data, selected }: TerminalNodeProps) {
  const { 
    terminal, 
    isSelected, 
    width, 
    height, 
    onNameChange, 
    onResize, 
    onContextMenu,
    onFocus,
    onStateChange 
  } = data
  
  const showAsSelected = selected || isSelected
  const isCollapsed = width < MIN_WIDTH || height < MIN_HEIGHT

  // State
  const [isEditingName, setIsEditingName] = useState(false)
  const [editedName, setEditedName] = useState(terminal.name)
  const [hasUnreadOutput, setHasUnreadOutput] = useState(false)

  // Refs
  const containerRef = useRef<HTMLDivElement>(null)
  const nameInputRef = useRef<HTMLInputElement>(null)

  // Terminal hook
  const {
    state,
    isConnecting,
    connect,
    disconnect,
    zoomIn,
    zoomOut,
    fit,
    copySelection,
    focus: focusTerminal,
    clear,
    sendInterrupt
  } = useTerminal(containerRef, terminal, {
    autoConnect: true,
    onFocus: () => {
      setHasUnreadOutput(false)
      onFocus?.(terminal.id)
    },
    onUnreadOutput: () => {
      setHasUnreadOutput(true)
    }
  })

  // Notify parent of state changes
  useEffect(() => {
    onStateChange?.(terminal.id, state)
  }, [terminal.id, state, onStateChange])

  // Focus name input when editing starts
  useEffect(() => {
    if (isEditingName && nameInputRef.current) {
      nameInputRef.current.focus()
      nameInputRef.current.select()
    }
  }, [isEditingName])

  // Fit terminal when size changes
  useEffect(() => {
    if (!isCollapsed) {
      fit()
    }
  }, [width, height, isCollapsed, fit])

  // Event handlers
  const handleContextMenu = useCallback((e: React.MouseEvent) => {
    e.preventDefault()
    e.stopPropagation()
    onContextMenu?.(e, terminal.id)
  }, [terminal.id, onContextMenu])

  const handleNameDoubleClick = useCallback((e: React.MouseEvent) => {
    e.stopPropagation()
    setIsEditingName(true)
    setEditedName(terminal.name)
  }, [terminal.name])

  const handleNameBlur = useCallback(() => {
    setIsEditingName(false)
    if (editedName !== terminal.name && editedName.trim()) {
      onNameChange?.(terminal.id, editedName.trim())
    }
  }, [terminal.id, terminal.name, editedName, onNameChange])

  const handleNameKeyDown = useCallback((e: React.KeyboardEvent) => {
    if (e.key === 'Enter') {
      handleNameBlur()
    } else if (e.key === 'Escape') {
      setIsEditingName(false)
      setEditedName(terminal.name)
    }
  }, [handleNameBlur, terminal.name])

  const handleResize = useCallback((_event: unknown, params: { width: number; height: number }) => {
    onResize?.(terminal.id, params.width, params.height)
  }, [terminal.id, onResize])

  const handleTerminalClick = useCallback(() => {
    focusTerminal()
  }, [focusTerminal])

  const handleConnect = useCallback(async () => {
    try {
      await connect()
    } catch (err) {
      console.error('Failed to connect terminal:', err)
    }
  }, [connect])

  const handleRestart = useCallback(async () => {
    await disconnect()
    await connect()
  }, [disconnect, connect])

  // Keyboard shortcuts
  const handleKeyDown = useCallback((e: React.KeyboardEvent) => {
    // Ctrl+Shift+C: Copy
    if (e.ctrlKey && e.shiftKey && e.key === 'C') {
      e.preventDefault()
      copySelection()
      return
    }
    // Ctrl+Shift+V: Paste
    if (e.ctrlKey && e.shiftKey && e.key === 'V') {
      e.preventDefault()
      navigator.clipboard.readText().then(text => {
        if (text) {
          // Write directly to terminal (not to IPC)
          // IPC write is handled by terminal onData
        }
      })
      return
    }
    // Ctrl+Plus: Zoom in
    if (e.ctrlKey && (e.key === '+' || e.key === '=')) {
      e.preventDefault()
      zoomIn()
      return
    }
    // Ctrl+Minus: Zoom out
    if (e.ctrlKey && e.key === '-') {
      e.preventDefault()
      zoomOut()
      return
    }
  }, [copySelection, zoomIn, zoomOut])

  const statusColors = STATUS_COLORS[state.status]

  // Collapsed view
  if (isCollapsed) {
    return (
      <>
        <NodeResizer
          isVisible={showAsSelected}
          minWidth={COLLAPSED_SIZE}
          minHeight={COLLAPSED_SIZE}
          onResize={handleResize}
          lineClassName="!border-gray-500"
          handleClassName="!bg-gray-600 !border-gray-700"
        />
        <div
          className={`rounded-lg border-2 ${statusColors.border} bg-gray-800 shadow-lg 
                     ${showAsSelected ? 'ring-2 ring-blue-400' : ''}
                     flex items-center justify-center cursor-pointer`}
          style={{ width: COLLAPSED_SIZE, height: COLLAPSED_SIZE }}
          onContextMenu={handleContextMenu}
          onClick={handleTerminalClick}
          title={`${terminal.name}\n${state.status}${terminal.metadata.lastCwd ? `\n${terminal.metadata.lastCwd}` : ''}`}
        >
          <span className="text-2xl">
            {state.status === 'running' ? '>' : state.status === 'error' ? '!' : '_'}
          </span>
          {hasUnreadOutput && (
            <span className="absolute top-1 right-1 w-2 h-2 rounded-full bg-blue-500 animate-pulse" />
          )}
        </div>
      </>
    )
  }

  // Full view
  return (
    <>
      <NodeResizer
        isVisible={showAsSelected}
        minWidth={MIN_WIDTH}
        minHeight={MIN_HEIGHT}
        onResize={handleResize}
        lineClassName="!border-gray-500"
        handleClassName="!bg-gray-600 !border-gray-700"
      />

      <div
        className={`rounded-lg border-2 ${statusColors.border} bg-gray-900 shadow-lg overflow-hidden
                   ${showAsSelected ? 'ring-2 ring-blue-400' : ''}`}
        style={{ width, height }}
        onContextMenu={handleContextMenu}
        onKeyDown={handleKeyDown}
      >
        {/* Header */}
        <div 
          className={`flex items-center gap-2 px-2 h-[${HEADER_HEIGHT}px] border-b border-gray-700 bg-gray-800 cursor-move`}
          style={{ height: HEADER_HEIGHT }}
        >
          {/* Status indicator */}
          <span 
            className={`w-2 h-2 rounded-full ${statusColors.indicator} ${state.status === 'running' ? 'animate-pulse' : ''}`}
            title={state.status}
          />

          {/* Terminal name */}
          {isEditingName ? (
            <input
              ref={nameInputRef}
              type="text"
              value={editedName}
              onChange={(e) => setEditedName(e.target.value)}
              onBlur={handleNameBlur}
              onKeyDown={handleNameKeyDown}
              className="flex-1 bg-gray-700 text-white text-sm px-1 rounded border border-gray-600 outline-none nodrag"
              maxLength={50}
            />
          ) : (
            <span
              className="flex-1 text-sm font-medium text-gray-200 truncate cursor-text nodrag"
              onDoubleClick={handleNameDoubleClick}
              title={terminal.name}
            >
              {terminal.name}
            </span>
          )}

          {/* Activity badge */}
          {hasUnreadOutput && (
            <span className="w-2 h-2 rounded-full bg-blue-500 animate-pulse" title="New output" />
          )}

          {/* CWD (truncated) */}
          {terminal.metadata.lastCwd && (
            <span 
              className="text-xs text-gray-500 truncate max-w-[150px]"
              title={terminal.metadata.lastCwd}
            >
              {terminal.metadata.lastCwd.split(/[/\\]/).pop()}
            </span>
          )}

          {/* Exit code for stopped/error */}
          {(state.status === 'stopped' || state.status === 'error') && state.exitCode !== undefined && (
            <span className={`text-xs ${state.status === 'error' ? 'text-red-400' : 'text-gray-500'}`}>
              exit: {state.exitCode}
            </span>
          )}
        </div>

        {/* Terminal content area */}
        <div 
          className="relative nodrag"
          style={{ height: height - HEADER_HEIGHT }}
          onClick={handleTerminalClick}
        >
          {/* XTerm container */}
          <div 
            ref={containerRef}
            className="w-full h-full"
          />

          {/* Overlay for disconnected/stopped/error states */}
          {state.status !== 'running' && (
            <div className={`absolute inset-0 flex flex-col items-center justify-center ${statusColors.bg} backdrop-blur-sm`}>
              {isConnecting ? (
                <div className="text-gray-300">
                  <span className="animate-spin inline-block w-5 h-5 border-2 border-current border-t-transparent rounded-full mr-2" />
                  Connecting...
                </div>
              ) : state.status === 'disconnected' ? (
                <>
                  <div className="text-gray-400 mb-2">Terminal disconnected</div>
                  <button
                    onClick={handleConnect}
                    className="px-4 py-2 bg-green-600 hover:bg-green-700 text-white rounded text-sm transition-colors"
                  >
                    Connect
                  </button>
                </>
              ) : state.status === 'error' ? (
                <>
                  <div className="text-red-400 mb-1">Terminal error</div>
                  {state.error && (
                    <div className="text-red-300 text-sm mb-2">{state.error}</div>
                  )}
                  <button
                    onClick={handleRestart}
                    className="px-4 py-2 bg-red-600 hover:bg-red-700 text-white rounded text-sm transition-colors"
                  >
                    Restart
                  </button>
                </>
              ) : (
                <>
                  <div className="text-gray-400 mb-2">
                    Terminal stopped {state.exitCode !== undefined && `(code: ${state.exitCode})`}
                  </div>
                  <div className="flex gap-2">
                    <button
                      onClick={handleRestart}
                      className="px-4 py-2 bg-blue-600 hover:bg-blue-700 text-white rounded text-sm transition-colors"
                    >
                      Restart
                    </button>
                  </div>
                </>
              )}
            </div>
          )}
        </div>
      </div>
    </>
  )
}

function TerminalNodeWithBoundary(props: TerminalNodeProps) {
  const handleRestart = useCallback(() => {
    // Force re-mount by updating key
    // This is handled by the parent component
  }, [])

  return (
    <TerminalErrorBoundary
      terminalId={props.data.terminal.id}
      terminalName={props.data.terminal.name}
      onRestart={handleRestart}
    >
      <TerminalNodeInner {...props} />
    </TerminalErrorBoundary>
  )
}

export const TerminalNode = memo(TerminalNodeWithBoundary)
