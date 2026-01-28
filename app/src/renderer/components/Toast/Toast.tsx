import React from 'react'
import { useAppStore } from '../../stores/appStore'

const toastStyles = {
  success: {
    bg: 'bg-green-50',
    border: 'border-green-200',
    text: 'text-green-800',
    icon: '✓'
  },
  error: {
    bg: 'bg-red-50',
    border: 'border-red-200',
    text: 'text-red-800',
    icon: '✕'
  },
  warning: {
    bg: 'bg-amber-50',
    border: 'border-amber-200',
    text: 'text-amber-800',
    icon: '⚠'
  },
  info: {
    bg: 'bg-blue-50',
    border: 'border-blue-200',
    text: 'text-blue-800',
    icon: 'ℹ'
  }
}

interface ToastItemProps {
  id: string
  type: 'success' | 'error' | 'info' | 'warning'
  message: string
}

function ToastItem({ id, type, message }: ToastItemProps) {
  const { removeToast } = useAppStore()
  const style = toastStyles[type]
  
  return (
    <div
      className={`flex items-center gap-3 px-4 py-3 rounded-lg shadow-lg border 
                 ${style.bg} ${style.border} ${style.text}
                 animate-in slide-in-from-right duration-300`}
      role="alert"
    >
      <span className="text-lg">{style.icon}</span>
      <span className="flex-1 text-sm">{message}</span>
      <button
        onClick={() => removeToast(id)}
        className="text-current opacity-60 hover:opacity-100 transition-opacity"
        aria-label="Dismiss"
      >
        ✕
      </button>
    </div>
  )
}

export function ToastContainer() {
  const { toasts } = useAppStore()
  
  if (toasts.length === 0) {
    return null
  }
  
  return (
    <div className="fixed bottom-4 right-4 z-50 flex flex-col gap-2 max-w-sm">
      {toasts.map((toast) => (
        <ToastItem key={toast.id} {...toast} />
      ))}
    </div>
  )
}

// Hook for easier toast usage
export function useToast() {
  const { addToast } = useAppStore()
  
  return {
    success: (message: string, duration?: number) => 
      addToast({ type: 'success', message, duration }),
    error: (message: string, duration?: number) => 
      addToast({ type: 'error', message, duration }),
    warning: (message: string, duration?: number) => 
      addToast({ type: 'warning', message, duration }),
    info: (message: string, duration?: number) => 
      addToast({ type: 'info', message, duration })
  }
}
