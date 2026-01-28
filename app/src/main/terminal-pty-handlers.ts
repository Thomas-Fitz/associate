/**
 * PTY IPC handlers for terminal operations.
 * 
 * Handles:
 * - pty:create - Spawn new PTY
 * - pty:write - Send input (max 64KB validation)
 * - pty:resize - Resize PTY (max 500x200 validation)
 * - pty:kill - Graceful shutdown
 * - pty:loadScrollback - Load scrollback from file
 */

import { ipcMain } from 'electron'
import { terminalManager } from './terminal-manager'
import type { TerminalConfig } from '../renderer/types/terminal'

export function setupPtyHandlers(): void {
  /**
   * Create a new PTY process for a terminal.
   */
  ipcMain.handle('pty:create', async (_event, terminalId: string, config: TerminalConfig) => {
    if (!terminalId || typeof terminalId !== 'string') {
      throw new Error('Invalid terminal ID')
    }
    
    await terminalManager.create(terminalId, config || {})
  })

  /**
   * Write data to a terminal's PTY.
   */
  ipcMain.handle('pty:write', async (_event, terminalId: string, data: string) => {
    if (!terminalId || typeof terminalId !== 'string') {
      throw new Error('Invalid terminal ID')
    }
    
    if (typeof data !== 'string') {
      throw new Error('Invalid data')
    }
    
    // Max input size is 64KB
    if (data.length > 64 * 1024) {
      throw new Error('Input too large (max 64KB)')
    }
    
    terminalManager.write(terminalId, data)
  })

  /**
   * Resize a terminal's PTY.
   */
  ipcMain.handle('pty:resize', async (_event, terminalId: string, cols: number, rows: number) => {
    if (!terminalId || typeof terminalId !== 'string') {
      throw new Error('Invalid terminal ID')
    }
    
    if (typeof cols !== 'number' || typeof rows !== 'number') {
      throw new Error('Invalid dimensions')
    }
    
    // Max dimensions are 500x200
    if (cols > 500 || rows > 200) {
      throw new Error('Dimensions too large (max 500x200)')
    }
    
    terminalManager.resize(terminalId, cols, rows)
  })

  /**
   * Kill a terminal's PTY.
   */
  ipcMain.handle('pty:kill', async (_event, terminalId: string) => {
    if (!terminalId || typeof terminalId !== 'string') {
      throw new Error('Invalid terminal ID')
    }
    
    await terminalManager.kill(terminalId)
  })

  /**
   * Load scrollback data for a terminal.
   */
  ipcMain.handle('pty:loadScrollback', async (_event, terminalId: string): Promise<string> => {
    if (!terminalId || typeof terminalId !== 'string') {
      throw new Error('Invalid terminal ID')
    }
    
    return terminalManager.loadScrollback(terminalId)
  })

  /**
   * Check if a terminal PTY is running.
   */
  ipcMain.handle('pty:isRunning', async (_event, terminalId: string): Promise<boolean> => {
    if (!terminalId || typeof terminalId !== 'string') {
      throw new Error('Invalid terminal ID')
    }
    
    return terminalManager.isRunning(terminalId)
  })

  /**
   * Get count of running terminals.
   */
  ipcMain.handle('pty:getRunningCount', async (): Promise<number> => {
    return terminalManager.getRunningCount()
  })
}
