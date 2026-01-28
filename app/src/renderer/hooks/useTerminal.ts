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

  // State
  const [fontSize, setFontSize] = useState(terminal.metadata.fontSize ?? DEFAULT_FONT_SIZE)
  const [isInitialized, setIsInitialized] = useState(false)

  // IPC hook
  const ipc = useTerminalIPC(terminal.id, terminal.config)

  // Initialize xterm.js
  useEffect(() => {
    if (!containerRef.current || xtermRef.current) {
      return
    }

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
    xterm.open(containerRef.current)
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
        onFocus?.()
      }
      textareaEl.onblur = () => {
        isFocusedRef.current = false
      }
    }

    // Handle terminal input → PTY write
    xterm.onData((data: string) => {
      ipc.write(data)
    })

    // Handle terminal resize → PTY resize
    xterm.onResize(({ cols, rows }: { cols: number; rows: number }) => {
      ipc.resize(cols, rows)
    })

    setIsInitialized(true)

    // Cleanup on unmount
    return () => {
      isMountedRef.current = false
      xterm.dispose()
      xtermRef.current = null
      fitAddonRef.current = null
      searchAddonRef.current = null
    }
  }, [containerRef, fontSize, ipc, onFocus])

  // Connect IPC data to xterm
  useEffect(() => {
    if (!isInitialized) return

    ipc.setOnData((data: string) => {
      xtermRef.current?.write(data)
      // Trigger unread output callback if terminal is not focused
      if (!isFocusedRef.current) {
        onUnreadOutput?.()
      }
    })

    ipc.setOnExit((exitCode: number, error?: string) => {
      const message = error 
        ? `\r\n\x1b[31mProcess exited with error: ${error} (code: ${exitCode})\x1b[0m\r\n`
        : `\r\n\x1b[90mProcess exited with code: ${exitCode}\x1b[0m\r\n`
      xtermRef.current?.write(message)
    })
  }, [isInitialized, ipc, onUnreadOutput])

  // Load scrollback and auto-connect
  useEffect(() => {
    if (!isInitialized) return

    const init = async () => {
      // Load previous scrollback
      const scrollback = await ipc.loadScrollback()
      if (scrollback && xtermRef.current) {
        xtermRef.current.write(scrollback)
      }

      // Check if already running
      const isRunning = await ipc.checkIsRunning()
      
      // Auto-connect if enabled and not already running
      if (autoConnect && !isRunning) {
        try {
          await ipc.connect()
        } catch (err) {
          console.error('Failed to auto-connect terminal:', err)
        }
      }
    }

    init()
  }, [isInitialized, autoConnect, ipc])

  // Fit terminal on container resize
  useEffect(() => {
    if (!containerRef.current || !fitAddonRef.current) return

    const resizeObserver = new ResizeObserver(() => {
      // Debounce the fit call
      requestAnimationFrame(() => {
        fitAddonRef.current?.fit()
      })
    })

    resizeObserver.observe(containerRef.current)

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
    ipc.write(text)
  }, [ipc])

  const clear = useCallback((): void => {
    xtermRef.current?.clear()
    // Also send clear command to shell
    if (process.platform === 'win32') {
      ipc.write('cls\r')
    } else {
      ipc.write('clear\r')
    }
  }, [ipc])

  const focus = useCallback((): void => {
    xtermRef.current?.focus()
  }, [])

  const sendInterrupt = useCallback((): void => {
    ipc.write('\x03') // Ctrl+C
  }, [ipc])

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
