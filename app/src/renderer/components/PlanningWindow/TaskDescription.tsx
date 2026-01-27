import React, { useState, useRef, useEffect, useCallback } from 'react'

interface TaskDescriptionProps {
  content: string
  onChange: (content: string) => void
}

export function TaskDescription({ content, onChange }: TaskDescriptionProps) {
  const [isEditing, setIsEditing] = useState(false)
  const [editValue, setEditValue] = useState(content)
  const textareaRef = useRef<HTMLTextAreaElement>(null)
  
  // Update edit value when content changes externally
  useEffect(() => {
    if (!isEditing) {
      setEditValue(content)
    }
  }, [content, isEditing])
  
  // Focus textarea when entering edit mode
  useEffect(() => {
    if (isEditing && textareaRef.current) {
      textareaRef.current.focus()
      textareaRef.current.select()
    }
  }, [isEditing])
  
  const handleSave = useCallback(() => {
    const trimmed = editValue.trim()
    if (trimmed !== content) {
      onChange(trimmed)
    }
    setIsEditing(false)
  }, [editValue, content, onChange])
  
  const handleCancel = useCallback(() => {
    setEditValue(content)
    setIsEditing(false)
  }, [content])
  
  const handleKeyDown = useCallback((e: React.KeyboardEvent) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault()
      handleSave()
    } else if (e.key === 'Escape') {
      e.preventDefault()
      handleCancel()
    }
  }, [handleSave, handleCancel])
  
  const handleClick = useCallback((e: React.MouseEvent) => {
    // Prevent triggering node drag
    e.stopPropagation()
    setIsEditing(true)
  }, [])
  
  if (isEditing) {
    return (
      <textarea
        ref={textareaRef}
        value={editValue}
        onChange={(e) => setEditValue(e.target.value)}
        onKeyDown={handleKeyDown}
        onBlur={handleSave}
        className="flex-1 p-2 text-sm resize-none border-none outline-none bg-white
                   focus:ring-2 focus:ring-primary-500 focus:ring-inset rounded"
        placeholder="Enter task description..."
        aria-label="Task description"
      />
    )
  }
  
  return (
    <div
      onClick={handleClick}
      className="flex-1 p-2 text-sm text-surface-700 cursor-text overflow-auto
                 hover:bg-surface-50 transition-colors"
      title="Click to edit"
    >
      {content || <span className="text-surface-400 italic">Click to add description...</span>}
    </div>
  )
}
