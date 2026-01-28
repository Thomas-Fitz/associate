import { useEffect, useRef, useCallback, useState, useMemo } from 'react'
import type { TerminalConfig, TerminalState } from '../types/terminal'

interface PtyDataEvent {
  terminalId: string
  data: string
}

interface PtyExitEvent {
  terminalId: string
  exitCode: number
  error?: string
}

/**
 * Hook for managing PTY IPC communication for a terminal.
 * Handles event subscriptions and provides methods for terminal operations.
 * 
 * This hook is responsible for:
 * - Subscribing to pty:data and pty:exit events for this terminal
 * - Providing write/resize/kill/connect operations
 * - Tracking terminal state (running/stopped/error/disconnected)
 * - Loading scrollback when reconnecting
 */
export function useTerminalIPC(terminalId: string, config: TerminalConfig = {}) {
  const [state, setState] = useState<TerminalState>({ status: 'disconnected' })
  const [isConnecting, setIsConnecting] = useState(false)
  
  // Track if PTY is ready to receive commands
  const isConnectedRef = useRef(false)
  
  // Callback refs for data and exit handlers
  const onDataRef = useRef<((data: string) => void) | null>(null)
  const onExitRef = useRef<((exitCode: number, error?: string) => void) | null>(null)
  
  // Stable config ref to avoid reconnecting on config object changes
  const configRef = useRef(config)
  configRef.current = config

  /**
   * Set the data handler (called when PTY outputs data)
   */
  const setOnData = useCallback((handler: (data: string) => void) => {
    onDataRef.current = handler
  }, [])

  /**
   * Set the exit handler (called when PTY exits)
   */
  const setOnExit = useCallback((handler: (exitCode: number, error?: string) => void) => {
    onExitRef.current = handler
  }, [])

  // Subscribe to PTY events on mount
  useEffect(() => {
    const handleData = (payload: PtyDataEvent) => {
      if (payload.terminalId === terminalId) {
        onDataRef.current?.(payload.data)
      }
    }

    const handleExit = (payload: PtyExitEvent) => {
      if (payload.terminalId === terminalId) {
        isConnectedRef.current = false
        const newState: TerminalState = {
          status: payload.error ? 'error' : 'stopped',
          exitCode: payload.exitCode,
          error: payload.error
        }
        setState(newState)
        onExitRef.current?.(payload.exitCode, payload.error)
      }
    }

    // Subscribe to events
    const unsubData = window.electronAPI.pty.onData(handleData)
    const unsubExit = window.electronAPI.pty.onExit(handleExit)

    // Cleanup subscriptions on unmount
    return () => {
      unsubData()
      unsubExit()
    }
  }, [terminalId])

  /**
   * Connect to the PTY (spawn a new shell process)
   */
  const connect = useCallback(async (): Promise<void> => {
    setIsConnecting(true)
    try {
      await window.electronAPI.pty.create(terminalId, configRef.current)
      isConnectedRef.current = true
      setState({ status: 'running' })
    } catch (err) {
      isConnectedRef.current = false
      setState({
        status: 'error',
        error: err instanceof Error ? err.message : 'Failed to connect'
      })
      throw err
    } finally {
      setIsConnecting(false)
    }
  }, [terminalId])

  /**
   * Disconnect from the PTY (kill the shell process)
   */
  const disconnect = useCallback(async (): Promise<void> => {
    try {
      isConnectedRef.current = false
      await window.electronAPI.pty.kill(terminalId)
      setState({ status: 'disconnected' })
    } catch (err) {
      console.error('Failed to disconnect terminal:', err)
    }
  }, [terminalId])

  /**
   * Write data to the PTY.
   * Silently ignores writes if PTY is not connected (prevents race conditions).
   */
  const write = useCallback((data: string): void => {
    if (!isConnectedRef.current) {
      // PTY not ready yet, ignore write
      return
    }
    window.electronAPI.pty.write(terminalId, data)
  }, [terminalId])

  /**
   * Resize the PTY.
   * Silently ignores resizes if PTY is not connected (prevents race conditions).
   */
  const resize = useCallback((cols: number, rows: number): void => {
    if (!isConnectedRef.current) {
      // PTY not ready yet, ignore resize
      return
    }
    window.electronAPI.pty.resize(terminalId, cols, rows)
  }, [terminalId])

  /**
   * Load scrollback from file
   */
  const loadScrollback = useCallback(async (): Promise<string> => {
    try {
      return await window.electronAPI.pty.loadScrollback(terminalId)
    } catch (err) {
      console.error('Failed to load scrollback:', err)
      return ''
    }
  }, [terminalId])

  /**
   * Check if the PTY is currently running.
   * Also syncs the isConnectedRef with the actual PTY state.
   */
  const checkIsRunning = useCallback(async (): Promise<boolean> => {
    try {
      const running = await window.electronAPI.pty.isRunning(terminalId)
      // Sync our connected state with the actual PTY state
      isConnectedRef.current = running
      if (running) {
        setState({ status: 'running' })
      }
      return running
    } catch (err) {
      isConnectedRef.current = false
      return false
    }
  }, [terminalId])

  /**
   * Mark the terminal as connected (used when PTY is already running on mount).
   */
  const markConnected = useCallback((): void => {
    isConnectedRef.current = true
    setState({ status: 'running' })
  }, [])

  // Return a memoized object to prevent unnecessary re-renders
  return useMemo(() => ({
    state,
    isConnecting,
    connect,
    disconnect,
    write,
    resize,
    loadScrollback,
    checkIsRunning,
    markConnected,
    setOnData,
    setOnExit
  }), [state, isConnecting, connect, disconnect, write, resize, loadScrollback, checkIsRunning, markConnected, setOnData, setOnExit])
}
