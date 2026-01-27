import React from 'react'
import { ConfirmDialog } from './ConfirmDialog'
import { useEdges } from '../../hooks'
import { useAppStore } from '../../stores/appStore'

export function DeleteEdgeDialog() {
  const { deleteEdgeDialog, hideDeleteEdgeDialog } = useAppStore()
  const { deleteEdges, getEdgeDescription } = useEdges()
  
  if (!deleteEdgeDialog?.visible || !deleteEdgeDialog.edges.length) {
    return null
  }
  
  const edgeCount = deleteEdgeDialog.edges.length
  const isMultiple = edgeCount > 1
  
  // Get edge description for single delete
  let edgeDescription = ''
  if (!isMultiple) {
    edgeDescription = getEdgeDescription(deleteEdgeDialog.edges[0])
  }
  
  const handleConfirm = async () => {
    await deleteEdges(deleteEdgeDialog.edges)
  }
  
  return (
    <ConfirmDialog
      title={isMultiple ? `Delete ${edgeCount} Relationships` : 'Delete Relationship'}
      message={
        isMultiple
          ? `Are you sure you want to delete ${edgeCount} relationships? This action cannot be undone.`
          : `Are you sure you want to delete this relationship? This action cannot be undone.${
              edgeDescription ? `\n\n${edgeDescription}` : ''
            }`
      }
      confirmLabel={isMultiple ? 'Delete All' : 'Delete'}
      danger
      onConfirm={handleConfirm}
      onCancel={hideDeleteEdgeDialog}
    />
  )
}
