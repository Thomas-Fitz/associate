import { useEffect, useRef, useCallback, useState, type RefObject } from 'react'
import { Terminal } from 'xterm'
import { FitAddon } from 'xterm-addon-fit'
import { SearchAddon } from 'xterm-addon-search'
import { WebLinksAddon } from 'xterm-addon-web-links'
import { useTerminalIPC } from './useTerminalIPC'
import type { TerminalInZone, TerminalState } from '../types/terminal'

// Default terminal dimensions
const DEFAULT_FONT_SIZE = 14
const MIN_FONT_SIZE = 8
const MAX_FONT_SIZE = 24

interface UseTerminalOptions {
  /** Auto-connect on mount */
  autoConnect?: boolean
  /** Callback when terminal focus changes */
  onFocus?: () => void
  /** Callback when terminal receives data while unfocused */
  onUnreadOutput?: () => void
}

interface UseTerminalReturn {
  /** xterm.js Terminal instance */
  xterm: Terminal | null
  /** Current terminal state */
  state: TerminalState
  /** Whether currently connecting */
  isConnecting: boolean
  /** Connect to PTY (spawn shell) */
  connect: () => Promise<void>
  /** Disconnect from PTY (kill shell) */
  disconnect: () => Promise<void>
  /** Search for text in terminal */
  search: (query: string, options?: { caseSensitive?: boolean }) => boolean
  /** Find next match */
  findNext: (query: string) => boolean
  /** Find previous match */
  findPrevious: (query: string) => boolean
  /** Clear search highlights */
  clearSearch: () => void
  /** Zoom in (increase font size) */
  zoomIn: () => void
  /** Zoom out (decrease font size) */
  zoomOut: () => void
  /** Reset zoom to default */
  resetZoom: () => void
  /** Current font size */
  fontSize: number
  /** Force fit terminal to container */
  fit: () => void
  /** Copy selected text to clipboard */
  copySelection: () => string
  /** Paste from clipboard */
  paste: (text: string) => void
  /** Clear the terminal screen */
  clear: () => void
  /** Focus the terminal */
  focus: () => void
  /** Send Ctrl+C (SIGINT) */
  sendInterrupt: () => void
}

/**
 * High-level hook for terminal management.
 * Integrates xterm.js with PTY IPC and manages the terminal lifecycle.
 * 
 * Features:
 * - XTerm.js instance lifecycle management
 * - Addon initialization (fit, webLinks, search)
 * - Connect IPC events to xterm
 * - Handle resize with fit addon
 * - Font size zoom (Ctrl+Plus/Minus)
 * - Search functionality (Ctrl+F)
 */
export function useTerminal(
  containerRef: RefObject<HTMLDivElement>,
  terminal: TerminalInZone,
  options: UseTerminalOptions = {}
): UseTerminalReturn {
  const { autoConnect = false, onFocus, onUnreadOutput } = options

  // Refs for xterm and addons
  const xtermRef = useRef<Terminal | null>(null)
  const fitAddonRef = useRef<FitAddon | null>(null)
  const searchAddonRef = useRef<SearchAddon | null>(null)
  const isMountedRef = useRef(true)
  const isFocusedRef = useRef(false)
  const hasInitializedRef = useRef(false)
  const hasLoadedScrollbackRef = useRef(false)

  // Stable refs for callbacks to avoid re-renders
  const onFocusRef = useRef(onFocus)
  const onUnreadOutputRef = useRef(onUnreadOutput)
  onFocusRef.current = onFocus
  onUnreadOutputRef.current = onUnreadOutput

  // State
  const [fontSize, setFontSize] = useState(terminal.metadata.fontSize ?? DEFAULT_FONT_SIZE)
  const [isInitialized, setIsInitialized] = useState(false)

  // IPC hook - stable because of useMemo in useTerminalIPC
  const ipc = useTerminalIPC(terminal.id, terminal.config)
  
  // Store IPC functions in refs for stable access in callbacks
  const ipcRef = useRef(ipc)
  ipcRef.current = ipc

  // Initialize xterm.js - only run once per terminal
  useEffect(() => {
    const container = containerRef.current
    if (!container || hasInitializedRef.current) {
      return
    }
    
    hasInitializedRef.current = true

    const xterm = new Terminal({
      fontSize,
      fontFamily: 'Consolas, "Courier New", monospace',
      theme: {
        background: '#1e1e1e',
        foreground: '#d4d4d4',
        cursor: '#d4d4d4',
        cursorAccent: '#1e1e1e',
        selectionBackground: '#264f78',
        black: '#000000',
        red: '#cd3131',
        green: '#0dbc79',
        yellow: '#e5e510',
        blue: '#2472c8',
        magenta: '#bc3fbc',
        cyan: '#11a8cd',
        white: '#e5e5e5',
        brightBlack: '#666666',
        brightRed: '#f14c4c',
        brightGreen: '#23d18b',
        brightYellow: '#f5f543',
        brightBlue: '#3b8eea',
        brightMagenta: '#d670d6',
        brightCyan: '#29b8db',
        brightWhite: '#ffffff'
      },
      cursorBlink: true,
      scrollback: 5000,
      allowProposedApi: true
    })

    // Initialize addons
    const fitAddon = new FitAddon()
    const searchAddon = new SearchAddon()
    const webLinksAddon = new WebLinksAddon((event: MouseEvent, uri: string) => {
      // Ctrl+Click to open links
      if (event.ctrlKey || event.metaKey) {
        window.open(uri, '_blank')
      }
    })

    xterm.loadAddon(fitAddon)
    xterm.loadAddon(searchAddon)
    xterm.loadAddon(webLinksAddon)

    // Open terminal in container
    xterm.open(container)
    fitAddon.fit()

    // Store refs
    xtermRef.current = xterm
    fitAddonRef.current = fitAddon
    searchAddonRef.current = searchAddon

    // Handle terminal focus - use the textarea's focus events
    const textareaEl = xterm.textarea
    if (textareaEl) {
      textareaEl.onfocus = () => {
        isFocusedRef.current = true
        onFocusRef.current?.()
      }
      textareaEl.onblur = () => {
        isFocusedRef.current = false
      }
    }

    // Handle terminal input → PTY write (use ref to get current ipc)
    xterm.onData((data: string) => {
      ipcRef.current.write(data)
    })

    // Handle terminal resize → PTY resize
    xterm.onResize(({ cols, rows }: { cols: number; rows: number }) => {
      ipcRef.current.resize(cols, rows)
    })

    setIsInitialized(true)

    // Cleanup on unmount
    return () => {
      isMountedRef.current = false
      hasInitializedRef.current = false
      hasLoadedScrollbackRef.current = false
      xterm.dispose()
      xtermRef.current = null
      fitAddonRef.current = null
      searchAddonRef.current = null
    }
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [terminal.id]) // Only re-run if terminal ID changes

  // Connect IPC data to xterm - set up once when initialized
  useEffect(() => {
    if (!isInitialized) return

    ipc.setOnData((data: string) => {
      xtermRef.current?.write(data)
      // Trigger unread output callback if terminal is not focused
      if (!isFocusedRef.current) {
        onUnreadOutputRef.current?.()
      }
    })

    ipc.setOnExit((exitCode: number, error?: string) => {
      const message = error 
        ? `\r\n\x1b[31mProcess exited with error: ${error} (code: ${exitCode})\x1b[0m\r\n`
        : `\r\n\x1b[90mProcess exited with code: ${exitCode}\x1b[0m\r\n`
      xtermRef.current?.write(message)
    })
  }, [isInitialized, ipc.setOnData, ipc.setOnExit])

  // Load scrollback and auto-connect - only run once
  useEffect(() => {
    if (!isInitialized || hasLoadedScrollbackRef.current) return

    hasLoadedScrollbackRef.current = true

    const init = async () => {
      // Load previous scrollback
      const scrollback = await ipcRef.current.loadScrollback()
      if (scrollback && xtermRef.current) {
        xtermRef.current.write(scrollback)
      }

      // Check if already running - this also syncs the connected state
      const isRunning = await ipcRef.current.checkIsRunning()
      
      // Auto-connect if enabled and not already running
      if (autoConnect && !isRunning) {
        try {
          await ipcRef.current.connect()
        } catch (err) {
          console.error('Failed to auto-connect terminal:', err)
          return // Don't try to resize if connect failed
        }
      }
      
      // Send initial resize now that PTY is ready
      // (resize events during xterm init were ignored because PTY wasn't connected yet)
      if (xtermRef.current && fitAddonRef.current) {
        fitAddonRef.current.fit()
        const { cols, rows } = xtermRef.current
        if (cols && rows) {
          ipcRef.current.resize(cols, rows)
        }
      }
    }

    init()
  }, [isInitialized, autoConnect])

  // Fit terminal on container resize
  useEffect(() => {
    const container = containerRef.current
    if (!container || !fitAddonRef.current) return

    const resizeObserver = new ResizeObserver(() => {
      // Debounce the fit call
      requestAnimationFrame(() => {
        fitAddonRef.current?.fit()
      })
    })

    resizeObserver.observe(container)

    return () => {
      resizeObserver.disconnect()
    }
  }, [containerRef])

  // Update font size in xterm when it changes
  useEffect(() => {
    if (xtermRef.current) {
      xtermRef.current.options.fontSize = fontSize
      fitAddonRef.current?.fit()
    }
  }, [fontSize])

  // Search functions
  const search = useCallback((query: string, opts?: { caseSensitive?: boolean }): boolean => {
    return searchAddonRef.current?.findNext(query, { 
      caseSensitive: opts?.caseSensitive ?? false 
    }) ?? false
  }, [])

  const findNext = useCallback((query: string): boolean => {
    return searchAddonRef.current?.findNext(query) ?? false
  }, [])

  const findPrevious = useCallback((query: string): boolean => {
    return searchAddonRef.current?.findPrevious(query) ?? false
  }, [])

  const clearSearch = useCallback((): void => {
    searchAddonRef.current?.clearDecorations()
  }, [])

  // Zoom functions
  const zoomIn = useCallback((): void => {
    setFontSize(prev => Math.min(prev + 1, MAX_FONT_SIZE))
  }, [])

  const zoomOut = useCallback((): void => {
    setFontSize(prev => Math.max(prev - 1, MIN_FONT_SIZE))
  }, [])

  const resetZoom = useCallback((): void => {
    setFontSize(DEFAULT_FONT_SIZE)
  }, [])

  // Utility functions
  const fit = useCallback((): void => {
    fitAddonRef.current?.fit()
  }, [])

  const copySelection = useCallback((): string => {
    const selection = xtermRef.current?.getSelection() ?? ''
    if (selection) {
      navigator.clipboard.writeText(selection)
    }
    return selection
  }, [])

  const paste = useCallback((text: string): void => {
    ipcRef.current.write(text)
  }, [])

  const clear = useCallback((): void => {
    xtermRef.current?.clear()
    // Also send clear command to shell
    if (process.platform === 'win32') {
      ipcRef.current.write('cls\r')
    } else {
      ipcRef.current.write('clear\r')
    }
  }, [])

  const focus = useCallback((): void => {
    xtermRef.current?.focus()
  }, [])

  const sendInterrupt = useCallback((): void => {
    ipcRef.current.write('\x03') // Ctrl+C
  }, [])

  return {
    xterm: xtermRef.current,
    state: ipc.state,
    isConnecting: ipc.isConnecting,
    connect: ipc.connect,
    disconnect: ipc.disconnect,
    search,
    findNext,
    findPrevious,
    clearSearch,
    zoomIn,
    zoomOut,
    resetZoom,
    fontSize,
    fit,
    copySelection,
    paste,
    focus,
    clear,
    sendInterrupt
  }
}
