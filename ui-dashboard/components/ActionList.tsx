'use client'

import { Action } from '@/lib/useStream'
import { approveAction, rejectAction } from '@/lib/api'
import { useState } from 'react'

interface Props {
  actions: Action[]
}

function formatActionType(actionType: string): string {
  return actionType
    .replace('ACTION_TYPE_', '')
    .toLowerCase()
    .replace(/_/g, ' ')
}

function formatStatus(status: string): string {
  return status.replace('ACTION_STATUS_', '').toLowerCase()
}

function getStatusClass(status: string): string {
  switch (status) {
    case 'ACTION_STATUS_PENDING':
      return 'bg-yellow-500/20 text-yellow-400'
    case 'ACTION_STATUS_APPROVED':
      return 'bg-blue-500/20 text-blue-400'
    case 'ACTION_STATUS_REJECTED':
      return 'bg-red-500/20 text-red-400'
    case 'ACTION_STATUS_EXECUTING':
      return 'bg-purple-500/20 text-purple-400'
    case 'ACTION_STATUS_COMPLETED':
      return 'bg-green-500/20 text-green-400'
    case 'ACTION_STATUS_FAILED':
      return 'bg-red-500/20 text-red-400'
    default:
      return 'bg-gray-500/20 text-gray-400'
  }
}

function formatTime(unixMs: string): string {
  return new Date(parseInt(unixMs)).toLocaleTimeString()
}

export function ActionList({ actions }: Props) {
  const [loading, setLoading] = useState<string | null>(null)

  const handleApprove = async (actionId: string) => {
    setLoading(actionId)
    try {
      await approveAction(actionId)
    } catch (e) {
      console.error('Failed to approve:', e)
    }
    setLoading(null)
  }

  const handleReject = async (actionId: string) => {
    setLoading(actionId)
    try {
      await rejectAction(actionId, 'Manually rejected')
    } catch (e) {
      console.error('Failed to reject:', e)
    }
    setLoading(null)
  }

  if (actions.length === 0) {
    return (
      <div className="text-center text-gray-500 py-8">
        No actions proposed
      </div>
    )
  }

  return (
    <div className="space-y-2">
      {actions.map((action) => (
        <div
          key={action.id.value}
          className="bg-gray-800/50 rounded-lg p-3 border border-gray-700"
        >
          <div className="flex items-start justify-between">
            <div className="flex-1">
              <div className="flex items-center gap-2">
                <span className={`text-xs px-2 py-0.5 rounded ${getStatusClass(action.status)}`}>
                  {formatStatus(action.status)}
                </span>
                <span className="text-xs text-gray-500">
                  {formatTime(action.createdAt.wallTimeUnixMs)}
                </span>
              </div>
              <h4 className="font-medium text-white mt-1 capitalize">
                {formatActionType(action.actionType)}
              </h4>
              <p className="text-sm text-gray-400 mt-1">{action.reason}</p>
              <p className="text-xs text-gray-500 mt-1">
                Target: {action.targetId.slice(0, 8)}...
              </p>
            </div>

            {action.status === 'ACTION_STATUS_PENDING' && (
              <div className="flex gap-2 ml-4">
                <button
                  onClick={() => handleApprove(action.id.value)}
                  disabled={loading === action.id.value}
                  className="px-3 py-1 bg-green-600 hover:bg-green-500 text-white text-sm rounded disabled:opacity-50"
                >
                  Approve
                </button>
                <button
                  onClick={() => handleReject(action.id.value)}
                  disabled={loading === action.id.value}
                  className="px-3 py-1 bg-red-600 hover:bg-red-500 text-white text-sm rounded disabled:opacity-50"
                >
                  Reject
                </button>
              </div>
            )}
          </div>
        </div>
      ))}
    </div>
  )
}
