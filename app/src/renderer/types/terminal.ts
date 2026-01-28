// Terminal types for zone terminals

export interface TerminalConfig {
  shell?: string                      // Override default shell path
  cwd?: string                        // Initial working directory
  env?: Record<string, string>        // Additional env vars to set
  envBlocklist?: string[]             // Env vars to exclude (default: ['ELECTRON_*'])
}

export interface TerminalState {
  status: 'running' | 'stopped' | 'error' | 'disconnected'
  exitCode?: number                   // Set when status is stopped/error
  error?: string                      // Error message if status is error
  hasUnreadOutput?: boolean           // True if new output since last focus
}

export interface TerminalInZone {
  id: string
  name: string                        // User-editable title ("Terminal 1", etc.)
  config: TerminalConfig
  state: TerminalState
  metadata: {
    ui_x?: number
    ui_y?: number
    ui_width?: number
    ui_height?: number
    ui_collapsed?: boolean            // True if resized below minimum
    lastCwd?: string                  // Last known working directory
    fontSize?: number                 // Terminal font size
    [key: string]: unknown
  }
  createdAt: string
  updatedAt: string
}

// Scrollback is NOT stored in metadata - stored in separate gzip file
// File path: app.getPath('userData')/terminals/{terminal-id}.gz

export interface TerminalCreateOptions {
  zoneId: string
  name?: string                       // Auto-generated if not provided
  config?: TerminalConfig
  metadata?: Record<string, unknown>  // UI position/size
}

export interface TerminalUpdateOptions {
  name?: string
  config?: Partial<TerminalConfig>
  state?: Partial<TerminalState>
  metadata?: Record<string, unknown>
}

// PTY events sent from main to renderer
export interface PtyDataEvent {
  terminalId: string
  data: string
}

export interface PtyExitEvent {
  terminalId: string
  exitCode: number
  error?: string
}
