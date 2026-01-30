'use client'

import { Service } from '@/lib/useStream'

interface Props {
  service: Service
}

function getHealthClass(health: string): string {
  switch (health) {
    case 'SERVICE_HEALTH_HEALTHY':
      return 'status-healthy'
    case 'SERVICE_HEALTH_DEGRADED':
      return 'status-degraded'
    case 'SERVICE_HEALTH_CRITICAL':
      return 'status-critical'
    case 'SERVICE_HEALTH_DOWN':
      return 'status-offline'
    default:
      return 'text-gray-400'
  }
}

function formatHealth(health: string): string {
  return health.replace('SERVICE_HEALTH_', '').toLowerCase()
}

export function ServiceCard({ service }: Props) {
  return (
    <div className="bg-gray-800/50 rounded-lg p-4 border border-gray-700">
      <div className="flex items-center justify-between mb-3">
        <h3 className="font-semibold text-white">{service.name}</h3>
        <span className={`text-sm capitalize ${getHealthClass(service.health)}`}>
          {formatHealth(service.health)}
        </span>
      </div>

      <div className="grid grid-cols-2 gap-2 text-sm">
        <div>
          <span className="text-gray-400">RPS</span>
          <p className="text-white font-mono">{service.requestsPerSecond.toFixed(0)}</p>
        </div>
        <div>
          <span className="text-gray-400">Error Rate</span>
          <p className={`font-mono ${service.errorRatePercent > 5 ? 'text-red-400' : 'text-white'}`}>
            {service.errorRatePercent.toFixed(2)}%
          </p>
        </div>
        <div>
          <span className="text-gray-400">P50 Latency</span>
          <p className="text-white font-mono">{service.latencyP50Ms.toFixed(1)}ms</p>
        </div>
        <div>
          <span className="text-gray-400">P99 Latency</span>
          <p className={`font-mono ${service.latencyP99Ms > 200 ? 'text-yellow-400' : 'text-white'}`}>
            {service.latencyP99Ms.toFixed(1)}ms
          </p>
        </div>
      </div>

      <div className="mt-3 pt-3 border-t border-gray-700 flex justify-between text-sm">
        <span className="text-gray-400">Replicas</span>
        <span className="text-gray-300">
          {service.replicaCount} / {service.desiredReplicas}
        </span>
      </div>
    </div>
  )
}
