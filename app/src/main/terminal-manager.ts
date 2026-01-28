/**
 * TerminalManager - Manages PTY processes for terminal instances.
 * 
 * Responsibilities:
 * - Spawn/destroy PTY processes
 * - Map terminal IDs to PTY instances
 * - Handle PTY output events → webContents.send
 * - Track running/stopped/error states
 * - Graceful shutdown (SIGTERM → 2s → taskkill/kill -9)
 * - Scrollback file I/O (gzip compression via zlib)
 */

import * as pty from 'node-pty'
import { app, WebContents } from 'electron'
import { join } from 'path'
import { promises as fs } from 'fs'
import { createGzip, createGunzip } from 'zlib'
import { createReadStream, createWriteStream } from 'fs'
import { pipeline } from 'stream/promises'
import type { TerminalConfig, TerminalState } from '../renderer/types/terminal'

// Maximum scrollback lines to persist
const MAX_SCROLLBACK_LINES = 5000
// Maximum input chunk size (64KB)
const MAX_INPUT_SIZE = 64 * 1024
// Maximum resize dimensions
const MAX_COLS = 500
const MAX_ROWS = 200
// Graceful shutdown timeout (ms)
const SHUTDOWN_TIMEOUT = 2000
// Scrollback save interval (ms)
const SAVE_INTERVAL = 30000
// Env vars to block by default
const DEFAULT_ENV_BLOCKLIST = ['ELECTRON_RUN_AS_NODE', 'ELECTRON_NO_ATTACH_CONSOLE']

interface PtyInstance {
  pty: pty.IPty
  terminalId: string
  scrollbackBuffer: string[]
  lastCwd: string
  saveTimer?: NodeJS.Timeout
}

export class TerminalManager {
  private instances: Map<string, PtyInstance> = new Map()
  private webContents: WebContents | null = null
  private terminalsDir: string

  constructor() {
    this.terminalsDir = join(app.getPath('userData'), 'terminals')
  }

  /**
   * Set the webContents to send PTY events to.
   */
  setWebContents(webContents: WebContents): void {
    this.webContents = webContents
  }

  /**
   * Get the terminals directory path.
   */
  getTerminalsDir(): string {
    return this.terminalsDir
  }

  /**
   * Ensure the terminals directory exists.
   */
  private async ensureTerminalsDir(): Promise<void> {
    try {
      await fs.mkdir(this.terminalsDir, { recursive: true })
    } catch (err) {
      console.error('Failed to create terminals directory:', err)
    }
  }

  /**
   * Get the scrollback file path for a terminal.
   */
  getScrollbackPath(terminalId: string): string {
    return join(this.terminalsDir, `${terminalId}.gz`)
  }

  /**
   * Get the default shell for the current platform.
   */
  private getDefaultShell(): string {
    if (process.platform === 'win32') {
      return process.env.COMSPEC || 'powershell.exe'
    }
    return process.env.SHELL || '/bin/bash'
  }

  /**
   * Filter environment variables based on blocklist.
   */
  private filterEnv(
    additionalEnv?: Record<string, string>,
    blocklist: string[] = DEFAULT_ENV_BLOCKLIST
  ): Record<string, string> {
    const env: Record<string, string> = {}
    
    // Copy process.env, filtering out blocked variables
    for (const [key, value] of Object.entries(process.env)) {
      if (value === undefined) continue
      
      // Check if key matches any blocklist pattern
      const isBlocked = blocklist.some(pattern => {
        if (pattern.endsWith('*')) {
          return key.startsWith(pattern.slice(0, -1))
        }
        return key === pattern
      })
      
      if (!isBlocked) {
        env[key] = value
      }
    }
    
    // Merge additional env vars
    if (additionalEnv) {
      Object.assign(env, additionalEnv)
    }
    
    return env
  }

  /**
   * Create a new PTY process for a terminal.
   */
  async create(terminalId: string, config: TerminalConfig): Promise<void> {
    // Check if already exists
    if (this.instances.has(terminalId)) {
      console.warn(`Terminal ${terminalId} already exists, killing existing instance`)
      await this.kill(terminalId)
    }

    await this.ensureTerminalsDir()

    const shell = config.shell || this.getDefaultShell()
    const cwd = config.cwd || app.getPath('home')
    const env = this.filterEnv(config.env, config.envBlocklist || DEFAULT_ENV_BLOCKLIST)

    // Determine shell args based on platform
    const shellArgs: string[] = []
    if (process.platform === 'win32' && shell.toLowerCase().includes('powershell')) {
      shellArgs.push('-NoLogo')
    }

    try {
      const ptyProcess = pty.spawn(shell, shellArgs, {
        name: 'xterm-256color',
        cols: 80,
        rows: 24,
        cwd,
        env
      })

      const instance: PtyInstance = {
        pty: ptyProcess,
        terminalId,
        scrollbackBuffer: [],
        lastCwd: cwd
      }

      // Set up data handler
      ptyProcess.onData((data: string) => {
        // Add to scrollback buffer
        const lines = data.split('\n')
        instance.scrollbackBuffer.push(...lines)
        
        // Trim scrollback to max lines
        if (instance.scrollbackBuffer.length > MAX_SCROLLBACK_LINES) {
          instance.scrollbackBuffer = instance.scrollbackBuffer.slice(-MAX_SCROLLBACK_LINES)
        }

        // Send to renderer
        if (this.webContents && !this.webContents.isDestroyed()) {
          this.webContents.send('pty:data', { terminalId, data })
        }
      })

      // Set up exit handler
      ptyProcess.onExit(({ exitCode, signal }: { exitCode: number; signal?: number }) => {
        console.log(`Terminal ${terminalId} exited with code ${exitCode}, signal ${signal}`)
        
        // Clear save timer
        if (instance.saveTimer) {
          clearInterval(instance.saveTimer)
        }

        // Save final scrollback
        this.saveScrollback(terminalId, instance.scrollbackBuffer.join('\n')).catch(console.error)

        // Send exit event to renderer
        if (this.webContents && !this.webContents.isDestroyed()) {
          const error = signal ? `Killed by signal ${signal}` : undefined
          this.webContents.send('pty:exit', { terminalId, exitCode: exitCode ?? 0, error })
        }

        // Remove from instances
        this.instances.delete(terminalId)
      })

      // Set up periodic scrollback save
      instance.saveTimer = setInterval(() => {
        this.saveScrollback(terminalId, instance.scrollbackBuffer.join('\n')).catch(console.error)
      }, SAVE_INTERVAL)

      this.instances.set(terminalId, instance)
      console.log(`Created terminal ${terminalId} with shell ${shell}`)
    } catch (err) {
      console.error(`Failed to create terminal ${terminalId}:`, err)
      
      // Send error to renderer
      if (this.webContents && !this.webContents.isDestroyed()) {
        this.webContents.send('pty:exit', {
          terminalId,
          exitCode: 1,
          error: err instanceof Error ? err.message : 'Failed to spawn shell'
        })
      }
      
      throw err
    }
  }

  /**
   * Write data to a terminal's PTY.
   */
  write(terminalId: string, data: string): void {
    const instance = this.instances.get(terminalId)
    if (!instance) {
      console.warn(`Terminal ${terminalId} not found`)
      return
    }

    // Validate input size
    if (data.length > MAX_INPUT_SIZE) {
      console.warn(`Input too large for terminal ${terminalId}: ${data.length} bytes`)
      return
    }

    instance.pty.write(data)
  }

  /**
   * Resize a terminal's PTY.
   */
  resize(terminalId: string, cols: number, rows: number): void {
    const instance = this.instances.get(terminalId)
    if (!instance) {
      console.warn(`Terminal ${terminalId} not found`)
      return
    }

    // Clamp dimensions
    const safeCols = Math.min(Math.max(1, Math.floor(cols)), MAX_COLS)
    const safeRows = Math.min(Math.max(1, Math.floor(rows)), MAX_ROWS)

    instance.pty.resize(safeCols, safeRows)
  }

  /**
   * Kill a terminal's PTY with graceful shutdown.
   */
  async kill(terminalId: string): Promise<void> {
    const instance = this.instances.get(terminalId)
    if (!instance) {
      console.warn(`Terminal ${terminalId} not found`)
      return
    }

    // Clear save timer
    if (instance.saveTimer) {
      clearInterval(instance.saveTimer)
    }

    // Save scrollback before killing
    await this.saveScrollback(terminalId, instance.scrollbackBuffer.join('\n'))

    // Try graceful shutdown first
    return new Promise<void>((resolve) => {
      const pid = instance.pty.pid

      // Set up timeout for force kill
      const forceKillTimer = setTimeout(() => {
        console.log(`Force killing terminal ${terminalId} (pid ${pid})`)
        
        if (process.platform === 'win32') {
          // Use taskkill on Windows
          const { spawn } = require('child_process')
          spawn('taskkill', ['/PID', String(pid), '/F', '/T'], { detached: true })
        } else {
          // Use SIGKILL on Unix
          try {
            process.kill(pid, 'SIGKILL')
          } catch (err) {
            console.error('Failed to force kill:', err)
          }
        }
        
        this.instances.delete(terminalId)
        resolve()
      }, SHUTDOWN_TIMEOUT)

      // Listen for exit
      const originalOnExit = instance.pty.onExit
      instance.pty.onExit(({ exitCode }: { exitCode: number }) => {
        clearTimeout(forceKillTimer)
        this.instances.delete(terminalId)
        resolve()
      })

      // Send graceful kill signal
      if (process.platform === 'win32') {
        // On Windows, we need to write Ctrl+C to the terminal
        instance.pty.write('\x03')
      } else {
        // On Unix, send SIGTERM
        try {
          process.kill(pid, 'SIGTERM')
        } catch (err) {
          console.error('Failed to send SIGTERM:', err)
          clearTimeout(forceKillTimer)
          this.instances.delete(terminalId)
          resolve()
        }
      }
    })
  }

  /**
   * Kill all PTY processes.
   */
  async killAll(): Promise<void> {
    const terminalIds = Array.from(this.instances.keys())
    console.log(`Killing ${terminalIds.length} terminals`)
    
    await Promise.all(terminalIds.map(id => this.kill(id)))
  }

  /**
   * Save scrollback data to a gzip file.
   */
  async saveScrollback(terminalId: string, data: string): Promise<void> {
    await this.ensureTerminalsDir()
    
    const filePath = this.getScrollbackPath(terminalId)
    
    try {
      const writeStream = createWriteStream(filePath)
      const gzip = createGzip()
      
      await new Promise<void>((resolve, reject) => {
        gzip.pipe(writeStream)
        gzip.write(data)
        gzip.end()
        
        writeStream.on('finish', resolve)
        writeStream.on('error', reject)
        gzip.on('error', reject)
      })
      
      console.log(`Saved scrollback for terminal ${terminalId}`)
    } catch (err) {
      console.error(`Failed to save scrollback for terminal ${terminalId}:`, err)
    }
  }

  /**
   * Load scrollback data from a gzip file.
   */
  async loadScrollback(terminalId: string): Promise<string> {
    const filePath = this.getScrollbackPath(terminalId)
    
    try {
      // Check if file exists
      await fs.access(filePath)
      
      const chunks: Buffer[] = []
      const readStream = createReadStream(filePath)
      const gunzip = createGunzip()
      
      await new Promise<void>((resolve, reject) => {
        readStream.pipe(gunzip)
        
        gunzip.on('data', (chunk: Buffer) => chunks.push(chunk))
        gunzip.on('end', resolve)
        gunzip.on('error', reject)
        readStream.on('error', reject)
      })
      
      return Buffer.concat(chunks).toString('utf-8')
    } catch (err) {
      // File doesn't exist or is corrupted
      console.log(`No scrollback found for terminal ${terminalId}`)
      return ''
    }
  }

  /**
   * Delete scrollback file for a terminal.
   */
  async deleteScrollbackFile(terminalId: string): Promise<void> {
    const filePath = this.getScrollbackPath(terminalId)
    
    try {
      await fs.unlink(filePath)
      console.log(`Deleted scrollback for terminal ${terminalId}`)
    } catch (err) {
      // File might not exist
      console.log(`No scrollback file to delete for terminal ${terminalId}`)
    }
  }

  /**
   * Check if a terminal is running.
   */
  isRunning(terminalId: string): boolean {
    return this.instances.has(terminalId)
  }

  /**
   * Get the count of running terminals.
   */
  getRunningCount(): number {
    return this.instances.size
  }

  /**
   * Get all running terminal IDs.
   */
  getRunningTerminalIds(): string[] {
    return Array.from(this.instances.keys())
  }
}

// Singleton instance
export const terminalManager = new TerminalManager()
