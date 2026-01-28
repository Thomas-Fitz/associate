import React from 'react'
import { ConfirmDialog } from './ConfirmDialog'
import { useAppStore } from '../../stores/appStore'
import { useDatabase, useZones } from '../../hooks'

export function DeleteMemoryDialog() {
  const { deleteMemoryDialog, hideDeleteMemoryDialog } = useAppStore()
  const db = useDatabase()
  const { refreshSelectedZone } = useZones()
  
  if (!deleteMemoryDialog?.visible) {
    return null
  }
  
  const handleConfirm = async () => {
    try {
      await db.memories.delete(deleteMemoryDialog.memoryId)
      refreshSelectedZone()
    } catch (err) {
      console.error('Failed to delete memory:', err)
    } finally {
      hideDeleteMemoryDialog()
    }
  }
  
  const typeName = deleteMemoryDialog.memoryType.toLowerCase()
  
  return (
    <ConfirmDialog
      title={`Delete ${deleteMemoryDialog.memoryType}`}
      message={`Are you sure you want to delete this ${typeName}? This action cannot be undone.`}
      confirmLabel="Delete"
      cancelLabel="Cancel"
      danger
      onConfirm={handleConfirm}
      onCancel={hideDeleteMemoryDialog}
    />
  )
}
