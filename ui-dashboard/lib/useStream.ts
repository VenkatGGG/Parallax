'use client'

import { useEffect, useState, useCallback } from 'react'

export interface MetricSnapshot {
  timestamp: {
    tickId: string
    wallTimeUnixMs: string
    simTimeUnixMs: string
  }
  nodes: Node[]
  services: Service[]
  traffic: TrafficStats
}

export interface Node {
  id: { value: string }
  name: string
  status: string
  cpuUsagePercent: number
  memoryUsagePercent: number
  diskUsagePercent: number
  runningServices: number
  availabilityZone: string
}

export interface Service {
  id: { value: string }
  name: string
  nodeId: { value: string }
  health: string
  requestsPerSecond: number
  errorRatePercent: number
  latencyP50Ms: number
  latencyP99Ms: number
  replicaCount: number
  desiredReplicas: number
}

export interface TrafficStats {
  totalRps: number
  totalErrorRate: number
  avgLatencyMs: number
  activeConnections: string
}

export interface Incident {
  id: { value: string }
  detectedAt: { tickId: string; wallTimeUnixMs: string }
  severity: string
  title: string
  description: string
  sourceService: string
  affectedIds: string[]
  ruleName: string
  metrics: Record<string, number>
  resolved: boolean
}

export interface Action {
  id: { value: string }
  incidentId: { value: string }
  proposedAtTick: string
  actionType: string
  targetId: string
  status: string
  reason: string
  parameters: Record<string, string>
  createdAt: { wallTimeUnixMs: string }
  resultMessage: string
}

interface StreamEvent {
  type: 'metrics' | 'incident' | 'action'
  payload: MetricSnapshot | Incident | Action
}

export function useStream(url: string) {
  const [connected, setConnected] = useState(false)
  const [metrics, setMetrics] = useState<MetricSnapshot | null>(null)
  const [incidents, setIncidents] = useState<Incident[]>([])
  const [actions, setActions] = useState<Action[]>([])

  useEffect(() => {
    const eventSource = new EventSource(url)

    eventSource.onopen = () => {
      setConnected(true)
    }

    eventSource.onmessage = (event) => {
      try {
        const data: StreamEvent = JSON.parse(event.data)

        switch (data.type) {
          case 'metrics':
            setMetrics(data.payload as MetricSnapshot)
            break
          case 'incident':
            setIncidents((prev) => {
              const incident = data.payload as Incident
              const exists = prev.some((i) => i.id.value === incident.id.value)
              if (exists) return prev
              return [incident, ...prev].slice(0, 50)
            })
            break
          case 'action':
            setActions((prev) => {
              const action = data.payload as Action
              const idx = prev.findIndex((a) => a.id.value === action.id.value)
              if (idx >= 0) {
                const updated = [...prev]
                updated[idx] = action
                return updated
              }
              return [action, ...prev].slice(0, 50)
            })
            break
        }
      } catch (e) {
        console.error('Failed to parse event:', e)
      }
    }

    eventSource.onerror = () => {
      setConnected(false)
    }

    return () => {
      eventSource.close()
    }
  }, [url])

  return { connected, metrics, incidents, actions }
}
