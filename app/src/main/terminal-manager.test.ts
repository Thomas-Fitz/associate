import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { promises as fs } from 'fs'
import { join } from 'path'
import { tmpdir } from 'os'

// Mock node-pty
vi.mock('node-pty', () => ({
  spawn: vi.fn(() => ({
    pid: 12345,
    onData: vi.fn((callback) => {
      // Store callback for later use
      return { dispose: vi.fn() }
    }),
    onExit: vi.fn((callback) => {
      return { dispose: vi.fn() }
    }),
    write: vi.fn(),
    resize: vi.fn(),
    kill: vi.fn()
  }))
}))

// Mock electron app
vi.mock('electron', () => ({
  app: {
    getPath: vi.fn((name: string) => {
      if (name === 'userData') return join(tmpdir(), 'associate-test')
      if (name === 'home') return tmpdir()
      return tmpdir()
    })
  },
  WebContents: class {}
}))

// Import after mocks
import { TerminalManager } from './terminal-manager'

describe('TerminalManager', () => {
  let manager: TerminalManager
  let testDir: string

  beforeEach(async () => {
    manager = new TerminalManager()
    testDir = manager.getTerminalsDir()
    // Ensure test directory exists
    await fs.mkdir(testDir, { recursive: true })
  })

  afterEach(async () => {
    // Cleanup test files
    try {
      const files = await fs.readdir(testDir)
      for (const file of files) {
        await fs.unlink(join(testDir, file))
      }
    } catch {
      // Directory might not exist
    }
  })

  describe('getScrollbackPath', () => {
    it('should return correct path for terminal scrollback', () => {
      const terminalId = 'test-terminal-123'
      const path = manager.getScrollbackPath(terminalId)
      
      expect(path).toContain('terminals')
      expect(path).toContain('test-terminal-123.gz')
    })
  })

  describe('saveScrollback and loadScrollback', () => {
    it('should save and load scrollback data', async () => {
      const terminalId = 'test-terminal-save-load'
      const testData = 'Hello, Terminal!\nLine 2\nLine 3'
      
      // Save scrollback
      await manager.saveScrollback(terminalId, testData)
      
      // Verify file exists
      const filePath = manager.getScrollbackPath(terminalId)
      await expect(fs.access(filePath)).resolves.toBeUndefined()
      
      // Load scrollback
      const loaded = await manager.loadScrollback(terminalId)
      
      expect(loaded).toBe(testData)
    })

    it('should return empty string when scrollback file does not exist', async () => {
      const terminalId = 'non-existent-terminal'
      
      const loaded = await manager.loadScrollback(terminalId)
      
      expect(loaded).toBe('')
    })

    it('should handle empty scrollback data', async () => {
      const terminalId = 'empty-scrollback'
      
      await manager.saveScrollback(terminalId, '')
      const loaded = await manager.loadScrollback(terminalId)
      
      expect(loaded).toBe('')
    })

    it('should handle scrollback with special characters', async () => {
      const terminalId = 'special-chars'
      const testData = 'Colors: \x1b[31mRed\x1b[0m \x1b[32mGreen\x1b[0m\nUnicode: '
      
      await manager.saveScrollback(terminalId, testData)
      const loaded = await manager.loadScrollback(terminalId)
      
      expect(loaded).toBe(testData)
    })

    it('should handle large scrollback data', async () => {
      const terminalId = 'large-scrollback'
      // Generate 1000 lines of text
      const lines = Array.from({ length: 1000 }, (_, i) => `Line ${i + 1}: ${'x'.repeat(80)}`)
      const testData = lines.join('\n')
      
      await manager.saveScrollback(terminalId, testData)
      const loaded = await manager.loadScrollback(terminalId)
      
      expect(loaded).toBe(testData)
    })
  })

  describe('deleteScrollbackFile', () => {
    it('should delete scrollback file when it exists', async () => {
      const terminalId = 'delete-test'
      const testData = 'Some scrollback data'
      
      // Create file first
      await manager.saveScrollback(terminalId, testData)
      
      // Verify it exists
      const filePath = manager.getScrollbackPath(terminalId)
      await expect(fs.access(filePath)).resolves.toBeUndefined()
      
      // Delete it
      await manager.deleteScrollbackFile(terminalId)
      
      // Verify it's gone
      await expect(fs.access(filePath)).rejects.toThrow()
    })

    it('should not throw when scrollback file does not exist', async () => {
      const terminalId = 'non-existent'
      
      // Should not throw
      await expect(manager.deleteScrollbackFile(terminalId)).resolves.toBeUndefined()
    })
  })

  describe('getRunningCount', () => {
    it('should return 0 when no terminals are running', () => {
      expect(manager.getRunningCount()).toBe(0)
    })
  })

  describe('isRunning', () => {
    it('should return false for non-existent terminal', () => {
      expect(manager.isRunning('non-existent')).toBe(false)
    })
  })

  describe('getRunningTerminalIds', () => {
    it('should return empty array when no terminals are running', () => {
      expect(manager.getRunningTerminalIds()).toEqual([])
    })
  })

  describe('scrollback persistence across sessions', () => {
    it('should persist scrollback that survives manager recreation', async () => {
      const terminalId = 'persist-test'
      const testData = 'Command output line 1\nCommand output line 2'
      
      // Save scrollback in first "session"
      await manager.saveScrollback(terminalId, testData)
      
      // Create a new manager instance (simulating app restart)
      const newManager = new TerminalManager()
      
      // Load scrollback in new "session"
      const loaded = await newManager.loadScrollback(terminalId)
      
      expect(loaded).toBe(testData)
    })

    it('should handle sequential save and load operations', async () => {
      const terminalId = 'sequential-test'
      
      // First save
      await manager.saveScrollback(terminalId, 'First content')
      expect(await manager.loadScrollback(terminalId)).toBe('First content')
      
      // Second save (overwrite)
      await manager.saveScrollback(terminalId, 'Second content')
      expect(await manager.loadScrollback(terminalId)).toBe('Second content')
      
      // Third save (overwrite)
      await manager.saveScrollback(terminalId, 'Third content')
      expect(await manager.loadScrollback(terminalId)).toBe('Third content')
    })

    it('should preserve scrollback with ANSI escape codes', async () => {
      const terminalId = 'ansi-test'
      const testData = [
        '\x1b[32mC:\\Users\\test>\x1b[0m dir',
        '\x1b[1m Volume in drive C is Windows\x1b[0m',
        '\x1b[33m Directory of C:\\Users\\test\x1b[0m',
        '',
        '\x1b[32m01/28/2026  07:30 AM\x1b[0m    <DIR>          Documents'
      ].join('\n')
      
      await manager.saveScrollback(terminalId, testData)
      const loaded = await manager.loadScrollback(terminalId)
      
      expect(loaded).toBe(testData)
    })

    it('should handle scrollback with Windows-style line endings', async () => {
      const terminalId = 'crlf-test'
      const testData = 'Line 1\r\nLine 2\r\nLine 3'
      
      await manager.saveScrollback(terminalId, testData)
      const loaded = await manager.loadScrollback(terminalId)
      
      expect(loaded).toBe(testData)
    })
  })
})
