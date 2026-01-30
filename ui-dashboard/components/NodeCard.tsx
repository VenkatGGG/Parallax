'use client'

import { Node } from '@/lib/useStream'

interface Props {
  node: Node
}

function getStatusClass(status: string): string {
  switch (status) {
    case 'NODE_STATUS_HEALTHY':
      return 'status-healthy'
    case 'NODE_STATUS_DEGRADED':
      return 'status-degraded'
    case 'NODE_STATUS_UNHEALTHY':
      return 'status-critical'
    case 'NODE_STATUS_OFFLINE':
      return 'status-offline'
    default:
      return 'text-gray-400'
  }
}

function formatStatus(status: string): string {
  return status.replace('NODE_STATUS_', '').toLowerCase()
}

export function NodeCard({ node }: Props) {
  return (
    <div className="bg-gray-800/50 rounded-lg p-4 border border-gray-700">
      <div className="flex items-center justify-between mb-3">
        <h3 className="font-semibold text-white">{node.name}</h3>
        <span className={`text-sm capitalize ${getStatusClass(node.status)}`}>
          {formatStatus(node.status)}
        </span>
      </div>

      <div className="space-y-2 text-sm">
        <div className="flex justify-between">
          <span className="text-gray-400">CPU</span>
          <div className="flex items-center gap-2">
            <div className="w-24 h-2 bg-gray-700 rounded-full overflow-hidden">
              <div
                className={`h-full rounded-full ${
                  node.cpuUsagePercent > 80 ? 'bg-red-500' : 'bg-green-500'
                }`}
                style={{ width: `${node.cpuUsagePercent}%` }}
              />
            </div>
            <span className="text-gray-300 w-12 text-right">
              {node.cpuUsagePercent.toFixed(1)}%
            </span>
          </div>
        </div>

        <div className="flex justify-between">
          <span className="text-gray-400">Memory</span>
          <div className="flex items-center gap-2">
            <div className="w-24 h-2 bg-gray-700 rounded-full overflow-hidden">
              <div
                className={`h-full rounded-full ${
                  node.memoryUsagePercent > 85 ? 'bg-red-500' : 'bg-blue-500'
                }`}
                style={{ width: `${node.memoryUsagePercent}%` }}
              />
            </div>
            <span className="text-gray-300 w-12 text-right">
              {node.memoryUsagePercent.toFixed(1)}%
            </span>
          </div>
        </div>

        <div className="flex justify-between">
          <span className="text-gray-400">Disk</span>
          <div className="flex items-center gap-2">
            <div className="w-24 h-2 bg-gray-700 rounded-full overflow-hidden">
              <div
                className="h-full rounded-full bg-purple-500"
                style={{ width: `${node.diskUsagePercent}%` }}
              />
            </div>
            <span className="text-gray-300 w-12 text-right">
              {node.diskUsagePercent.toFixed(1)}%
            </span>
          </div>
        </div>

        <div className="flex justify-between pt-2 border-t border-gray-700">
          <span className="text-gray-400">Services</span>
          <span className="text-gray-300">{node.runningServices}</span>
        </div>
        <div className="flex justify-between">
          <span className="text-gray-400">Zone</span>
          <span className="text-gray-300">{node.availabilityZone}</span>
        </div>
      </div>
    </div>
  )
}
