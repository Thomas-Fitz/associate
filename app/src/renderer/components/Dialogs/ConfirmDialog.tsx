import React, { useEffect, useRef } from 'react'

interface ConfirmDialogProps {
  title: string
  message: string
  confirmLabel?: string
  cancelLabel?: string
  danger?: boolean
  onConfirm: () => void
  onCancel: () => void
}

export function ConfirmDialog({
  title,
  message,
  confirmLabel = 'Confirm',
  cancelLabel = 'Cancel',
  danger = false,
  onConfirm,
  onCancel
}: ConfirmDialogProps) {
  const confirmButtonRef = useRef<HTMLButtonElement>(null)
  
  // Focus confirm button on mount
  useEffect(() => {
    confirmButtonRef.current?.focus()
  }, [])
  
  // Handle keyboard events
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        onCancel()
      } else if (e.key === 'Enter') {
        onConfirm()
      }
    }
    
    document.addEventListener('keydown', handleKeyDown)
    return () => document.removeEventListener('keydown', handleKeyDown)
  }, [onConfirm, onCancel])
  
  return (
    <div className="dialog-overlay" onClick={onCancel} role="dialog" aria-modal="true" aria-labelledby="dialog-title">
      <div className="dialog-content" onClick={(e) => e.stopPropagation()}>
        <div className="p-4 border-b border-surface-200">
          <h2 id="dialog-title" className="text-lg font-semibold flex items-center gap-2">
            {danger && <span className="text-amber-500">&#9888;</span>}
            {title}
          </h2>
        </div>
        
        <div className="p-4">
          <p className="text-surface-700">{message}</p>
        </div>
        
        <div className="p-4 border-t border-surface-200 flex justify-end gap-3">
          <button
            onClick={onCancel}
            className="px-4 py-2 text-sm border border-surface-300 rounded-md hover:bg-surface-100 transition-colors"
          >
            {cancelLabel}
          </button>
          <button
            ref={confirmButtonRef}
            onClick={onConfirm}
            className={`px-4 py-2 text-sm text-white rounded-md transition-colors ${
              danger 
                ? 'bg-red-600 hover:bg-red-700' 
                : 'bg-primary-600 hover:bg-primary-700'
            }`}
          >
            {confirmLabel}
          </button>
        </div>
      </div>
    </div>
  )
}
