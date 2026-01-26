import React, { useEffect, useRef } from 'react'

interface ContextMenuProps {
  x: number
  y: number
  onClose: () => void
  children: React.ReactNode
}

export function ContextMenu({ x, y, onClose, children }: ContextMenuProps) {
  const menuRef = useRef<HTMLDivElement>(null)
  
  // Close on click outside
  useEffect(() => {
    const handleClickOutside = (e: MouseEvent) => {
      if (menuRef.current && !menuRef.current.contains(e.target as Node)) {
        onClose()
      }
    }
    
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        onClose()
      }
    }
    
    // Delay adding listeners to avoid immediate close
    setTimeout(() => {
      document.addEventListener('mousedown', handleClickOutside)
      document.addEventListener('keydown', handleKeyDown)
    }, 0)
    
    return () => {
      document.removeEventListener('mousedown', handleClickOutside)
      document.removeEventListener('keydown', handleKeyDown)
    }
  }, [onClose])
  
  // Adjust position to stay within viewport
  const adjustedPosition = {
    x: Math.min(x, window.innerWidth - 200),
    y: Math.min(y, window.innerHeight - 200)
  }
  
  return (
    <div
      ref={menuRef}
      className="context-menu"
      style={{
        left: adjustedPosition.x,
        top: adjustedPosition.y
      }}
      role="menu"
      aria-label="Context menu"
    >
      {children}
    </div>
  )
}

interface ContextMenuItemProps {
  onClick: () => void
  disabled?: boolean
  danger?: boolean
  children: React.ReactNode
}

export function ContextMenuItem({ onClick, disabled, danger, children }: ContextMenuItemProps) {
  return (
    <button
      onClick={disabled ? undefined : onClick}
      className={`context-menu-item w-full text-left ${danger ? 'danger' : ''} ${disabled ? 'opacity-50 cursor-not-allowed' : ''}`}
      role="menuitem"
      disabled={disabled}
    >
      {children}
    </button>
  )
}

export function ContextMenuSeparator() {
  return <div className="context-menu-separator" role="separator" />
}

interface ContextMenuSubMenuProps {
  label: string
  children: React.ReactNode
}

export function ContextMenuSubMenu({ label, children }: ContextMenuSubMenuProps) {
  const [isOpen, setIsOpen] = React.useState(false)
  
  return (
    <div
      className="relative"
      onMouseEnter={() => setIsOpen(true)}
      onMouseLeave={() => setIsOpen(false)}
    >
      <div className="context-menu-item flex items-center justify-between">
        <span>{label}</span>
        <span className="text-surface-400 ml-4">&#9656;</span>
      </div>
      
      {isOpen && (
        <div className="absolute left-full top-0 ml-0.5 context-menu">
          {children}
        </div>
      )}
    </div>
  )
}
