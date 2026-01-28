import React from 'react'
import { ConfirmDialog } from './ConfirmDialog'
import { useAppStore } from '../../stores/appStore'
import { useDatabase, useZones } from '../../hooks'

export function DeletePlanDialog() {
  const { deletePlanDialog, hideDeletePlanDialog } = useAppStore()
  const db = useDatabase()
  const { refreshSelectedZone } = useZones()
  
  if (!deletePlanDialog?.visible) {
    return null
  }
  
  const handleConfirm = async () => {
    try {
      await db.plans.delete(deletePlanDialog.planId)
      refreshSelectedZone()
    } catch (err) {
      console.error('Failed to delete plan:', err)
    } finally {
      hideDeletePlanDialog()
    }
  }
  
  const taskCountText = deletePlanDialog.taskCount > 0
    ? ` This will also delete ${deletePlanDialog.taskCount} task${deletePlanDialog.taskCount === 1 ? '' : 's'} in this plan.`
    : ''
  
  return (
    <ConfirmDialog
      title="Delete Plan"
      message={`Are you sure you want to delete "${deletePlanDialog.planName}"?${taskCountText} This action cannot be undone.`}
      confirmLabel="Delete Plan"
      cancelLabel="Cancel"
      danger
      onConfirm={handleConfirm}
      onCancel={hideDeletePlanDialog}
    />
  )
}
