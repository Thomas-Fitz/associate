import type { ElectronAPI } from '../../preload'

declare global {
  interface Window {
    electronAPI: ElectronAPI
  }
}

export function useDatabase() {
  // Access the electron API exposed via preload
  const api = window.electronAPI
  
  if (!api) {
    console.warn('Electron API not available - running in browser mode')
    // Return mock API for development in browser
    return {
      plans: {
        list: async () => [],
        get: async () => null
      },
      tasks: {
        create: async () => { throw new Error('Not available in browser mode') },
        update: async () => { throw new Error('Not available in browser mode') },
        delete: async () => { throw new Error('Not available in browser mode') },
        reorder: async () => { throw new Error('Not available in browser mode') }
      },
      dependencies: {
        create: async () => { throw new Error('Not available in browser mode') },
        delete: async () => { throw new Error('Not available in browser mode') }
      }
    }
  }
  
  return api
}
