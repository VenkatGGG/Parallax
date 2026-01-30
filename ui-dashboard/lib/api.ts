const API_BASE = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8081'

export async function approveAction(actionId: string): Promise<{ success: boolean; message: string }> {
  const response = await fetch(`${API_BASE}/ops.v1.ActionService/ApproveAction`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      'Connect-Protocol-Version': '1',
    },
    body: JSON.stringify({
      actionId: { value: actionId },
    }),
  })

  if (!response.ok) {
    throw new Error(`Failed to approve action: ${response.statusText}`)
  }

  return response.json()
}

export async function rejectAction(actionId: string, reason: string): Promise<{ success: boolean }> {
  const response = await fetch(`${API_BASE}/ops.v1.ActionService/RejectAction`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      'Connect-Protocol-Version': '1',
    },
    body: JSON.stringify({
      actionId: { value: actionId },
      reason,
    }),
  })

  if (!response.ok) {
    throw new Error(`Failed to reject action: ${response.statusText}`)
  }

  return response.json()
}

export async function listPendingActions(limit: number = 50) {
  const response = await fetch(`${API_BASE}/ops.v1.ActionService/ListPendingActions`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      'Connect-Protocol-Version': '1',
    },
    body: JSON.stringify({ limit }),
  })

  if (!response.ok) {
    throw new Error(`Failed to list actions: ${response.statusText}`)
  }

  return response.json()
}

export async function getActionHistory(limit: number = 100) {
  const response = await fetch(`${API_BASE}/ops.v1.ActionService/GetActionHistory`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      'Connect-Protocol-Version': '1',
    },
    body: JSON.stringify({ limit }),
  })

  if (!response.ok) {
    throw new Error(`Failed to get action history: ${response.statusText}`)
  }

  return response.json()
}
