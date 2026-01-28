import React, { Component, type ReactNode, type ErrorInfo } from 'react'

interface TerminalErrorBoundaryProps {
  children: ReactNode
  terminalId: string
  terminalName: string
  onRestart?: () => void
}

interface TerminalErrorBoundaryState {
  hasError: boolean
  error: Error | null
}

/**
 * Error boundary for terminal nodes.
 * Catches render errors in TerminalNode and displays a fallback UI
 * that allows the user to restart the terminal.
 */
export class TerminalErrorBoundary extends Component<TerminalErrorBoundaryProps, TerminalErrorBoundaryState> {
  constructor(props: TerminalErrorBoundaryProps) {
    super(props)
    this.state = { hasError: false, error: null }
  }

  static getDerivedStateFromError(error: Error): TerminalErrorBoundaryState {
    return { hasError: true, error }
  }

  componentDidCatch(error: Error, errorInfo: ErrorInfo): void {
    console.error(`Terminal error [${this.props.terminalId}]:`, error, errorInfo)
  }

  handleRestart = (): void => {
    this.setState({ hasError: false, error: null })
    this.props.onRestart?.()
  }

  render(): ReactNode {
    if (this.state.hasError) {
      return (
        <div className="flex flex-col items-center justify-center h-full bg-gray-900 text-gray-300 p-4 rounded-lg border-2 border-red-500">
          <div className="text-red-400 text-lg font-semibold mb-2">
            Terminal Error
          </div>
          <div className="text-sm text-gray-400 mb-4 text-center">
            {this.props.terminalName} encountered an error
          </div>
          {this.state.error && (
            <div className="text-xs text-gray-500 mb-4 max-w-full overflow-hidden text-ellipsis">
              {this.state.error.message}
            </div>
          )}
          <button
            onClick={this.handleRestart}
            className="px-4 py-2 bg-red-600 hover:bg-red-700 text-white rounded text-sm transition-colors"
          >
            Restart Terminal
          </button>
        </div>
      )
    }

    return this.props.children
  }
}
