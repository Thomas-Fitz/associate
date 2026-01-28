import { app, BrowserWindow, shell } from 'electron'
import { join } from 'path'
import { initializeDatabase, closeDatabase } from './database'
import { setupIpcHandlers } from './ipc-handlers'
import { setupPtyHandlers } from './terminal-pty-handlers'
import { terminalManager } from './terminal-manager'

let mainWindow: BrowserWindow | null = null

function createWindow(): void {
  mainWindow = new BrowserWindow({
    width: 1400,
    height: 900,
    minWidth: 800,
    minHeight: 600,
    show: false,
    autoHideMenuBar: false,
    webPreferences: {
      preload: join(__dirname, '../preload/index.js'),
      sandbox: false,
      contextIsolation: true,
      nodeIntegration: false
    }
  })

  mainWindow.on('ready-to-show', () => {
    mainWindow?.show()
  })

  // Set webContents for terminal manager
  terminalManager.setWebContents(mainWindow.webContents)

  mainWindow.webContents.setWindowOpenHandler((details) => {
    shell.openExternal(details.url)
    return { action: 'deny' }
  })

  // Load the renderer
  if (process.env.ELECTRON_RENDERER_URL) {
    mainWindow.loadURL(process.env.ELECTRON_RENDERER_URL)
  } else {
    mainWindow.loadFile(join(__dirname, '../renderer/index.html'))
  }
}

// Initialize the app
app.whenReady().then(async () => {
  // Initialize database connection
  try {
    await initializeDatabase()
    console.log('Database connection established')
  } catch (err) {
    console.error('Failed to initialize database:', err)
    // Continue anyway - we'll show error state in UI
  }

  // Set up IPC handlers
  setupIpcHandlers()
  
  // Set up PTY handlers
  setupPtyHandlers()

  // Create the main window
  createWindow()

  app.on('activate', () => {
    if (BrowserWindow.getAllWindows().length === 0) {
      createWindow()
    }
  })
})

// Quit when all windows are closed (except on macOS)
app.on('window-all-closed', () => {
  if (process.platform !== 'darwin') {
    app.quit()
  }
})

// Clean up database connection and terminals on quit
app.on('before-quit', async (event) => {
  const runningCount = terminalManager.getRunningCount()
  
  if (runningCount > 0) {
    console.log(`Cleaning up ${runningCount} running terminals...`)
    await terminalManager.killAll()
  }
  
  await closeDatabase()
})
