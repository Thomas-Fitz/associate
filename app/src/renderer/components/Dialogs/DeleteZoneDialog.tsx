import React from 'react'
import { ConfirmDialog } from './ConfirmDialog'
import { useAppStore } from '../../stores/appStore'
import { useZones } from '../../hooks'

export function DeleteZoneDialog() {
  const { deleteZoneDialog, hideDeleteZoneDialog } = useAppStore()
  const { deleteZone, selectedZoneId, selectZone } = useZones()
  
  if (!deleteZoneDialog?.visible) {
    return null
  }
  
  const handleConfirm = async () => {
    try {
      // If deleting the selected zone, clear selection first
      if (selectedZoneId === deleteZoneDialog.zoneId) {
        selectZone(null)
      }
      await deleteZone(deleteZoneDialog.zoneId)
    } catch (err) {
      console.error('Failed to delete zone:', err)
    } finally {
      hideDeleteZoneDialog()
    }
  }
  
  return (
    <ConfirmDialog
      title="Delete Zone"
      message={`Are you sure you want to delete "${deleteZoneDialog.zoneName}"? This will permanently delete all plans, tasks, and memories in this zone. This action cannot be undone.`}
      confirmLabel="Delete Zone"
      cancelLabel="Cancel"
      danger
      onConfirm={handleConfirm}
      onCancel={hideDeleteZoneDialog}
    />
  )
}
