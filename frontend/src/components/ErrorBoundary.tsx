import React, { Component, ErrorInfo, ReactNode } from 'react'
import { AlertTriangle, RefreshCw } from 'lucide-react'

interface Props {
  children: ReactNode
}

interface State {
  hasError: boolean
  error: Error | null
  errorInfo: ErrorInfo | null
}

export class ErrorBoundary extends Component<Props, State> {
  public state: State = {
    hasError: false,
    error: null,
    errorInfo: null,
  }

  public static getDerivedStateFromError(error: Error): State {
    return { hasError: true, error, errorInfo: null }
  }

  public componentDidCatch(error: Error, errorInfo: ErrorInfo) {
    console.error('Lumina Uncaught React Error:', error, errorInfo)
    this.setState({ error, errorInfo })
  }

  private handleReload = () => {
    window.location.reload()
  }

  public render() {
    if (this.state.hasError) {
      return (
        <div className="h-screen w-screen bg-slate-950 flex items-center justify-center p-6 text-slate-100 select-none">
          <div className="max-w-lg w-full p-8 rounded-2xl bg-slate-900/90 border border-rose-500/30 shadow-2xl backdrop-blur-2xl text-center space-y-4">
            <div className="w-14 h-14 rounded-2xl bg-rose-500/20 text-rose-400 mx-auto flex items-center justify-center">
              <AlertTriangle className="w-8 h-8" />
            </div>

            <h2 className="text-xl font-bold text-white tracking-tight">Something went wrong</h2>
            <p className="text-xs text-slate-400">
              Lumina encountered an unexpected interface error. A visual crash report is provided below:
            </p>

            <div className="p-3 rounded-xl bg-black/50 border border-white/5 text-left text-xs font-mono text-rose-300 max-h-36 overflow-y-auto whitespace-pre-wrap">
              {this.state.error?.toString()}
              {this.state.errorInfo?.componentStack}
            </div>

            <button
              onClick={this.handleReload}
              className="px-6 py-2.5 rounded-full bg-rose-600 hover:bg-rose-500 text-white text-xs font-semibold shadow-lg shadow-rose-600/30 transition flex items-center gap-2 mx-auto"
            >
              <RefreshCw className="w-4 h-4" />
              <span>Reload Lumina Interface</span>
            </button>
          </div>
        </div>
      )
    }

    return this.props.children
  }
}
