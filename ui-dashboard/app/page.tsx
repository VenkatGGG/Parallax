'use client'

import { useStream } from '@/lib/useStream'
import { NodeCard } from '@/components/NodeCard'
import { ServiceCard } from '@/components/ServiceCard'
import { IncidentList } from '@/components/IncidentList'
import { ActionList } from '@/components/ActionList'

const STREAM_URL = process.env.NEXT_PUBLIC_STREAM_URL || 'http://localhost:8081/api/stream'

export default function Dashboard() {
  const { connected, metrics, incidents, actions } = useStream(STREAM_URL)

  return (
    <div className="min-h-screen p-6">
      <header className="mb-8">
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-3xl font-bold text-white">Microcloud Dashboard</h1>
            <p className="text-gray-400 mt-1">Self-Healing Simulation Monitor</p>
          </div>
          <div className="flex items-center gap-4">
            {metrics && (
              <div className="text-right">
                <p className="text-sm text-gray-400">Tick</p>
                <p className="text-xl font-mono text-white">{metrics.timestamp.tickId}</p>
              </div>
            )}
            <div className={`w-3 h-3 rounded-full ${connected ? 'bg-green-500' : 'bg-red-500'}`} />
          </div>
        </div>
      </header>

      {metrics && (
        <div className="grid grid-cols-4 gap-4 mb-8">
          <div className="bg-gray-800/50 rounded-lg p-4 border border-gray-700">
            <p className="text-sm text-gray-400">Total RPS</p>
            <p className="text-2xl font-mono text-white">
              {metrics.traffic.totalRps.toFixed(0)}
            </p>
          </div>
          <div className="bg-gray-800/50 rounded-lg p-4 border border-gray-700">
            <p className="text-sm text-gray-400">Error Rate</p>
            <p className={`text-2xl font-mono ${metrics.traffic.totalErrorRate > 5 ? 'text-red-400' : 'text-white'}`}>
              {metrics.traffic.totalErrorRate.toFixed(2)}%
            </p>
          </div>
          <div className="bg-gray-800/50 rounded-lg p-4 border border-gray-700">
            <p className="text-sm text-gray-400">Avg Latency</p>
            <p className="text-2xl font-mono text-white">
              {metrics.traffic.avgLatencyMs.toFixed(1)}ms
            </p>
          </div>
          <div className="bg-gray-800/50 rounded-lg p-4 border border-gray-700">
            <p className="text-sm text-gray-400">Connections</p>
            <p className="text-2xl font-mono text-white">
              {metrics.traffic.activeConnections}
            </p>
          </div>
        </div>
      )}

      <div className="grid grid-cols-3 gap-6">
        <div className="col-span-2 space-y-6">
          <section>
            <h2 className="text-xl font-semibold text-white mb-4">Nodes</h2>
            <div className="grid grid-cols-3 gap-4">
              {metrics?.nodes.map((node) => (
                <NodeCard key={node.id.value} node={node} />
              ))}
            </div>
          </section>

          <section>
            <h2 className="text-xl font-semibold text-white mb-4">Services</h2>
            <div className="grid grid-cols-3 gap-4">
              {metrics?.services.map((service) => (
                <ServiceCard key={service.id.value} service={service} />
              ))}
            </div>
          </section>
        </div>

        <div className="space-y-6">
          <section>
            <h2 className="text-xl font-semibold text-white mb-4">
              Incidents
              {incidents.length > 0 && (
                <span className="ml-2 text-sm bg-red-500/20 text-red-400 px-2 py-0.5 rounded">
                  {incidents.length}
                </span>
              )}
            </h2>
            <div className="max-h-[400px] overflow-y-auto">
              <IncidentList incidents={incidents} />
            </div>
          </section>

          <section>
            <h2 className="text-xl font-semibold text-white mb-4">
              Actions
              {actions.filter(a => a.status === 'ACTION_STATUS_PENDING').length > 0 && (
                <span className="ml-2 text-sm bg-yellow-500/20 text-yellow-400 px-2 py-0.5 rounded">
                  {actions.filter(a => a.status === 'ACTION_STATUS_PENDING').length} pending
                </span>
              )}
            </h2>
            <div className="max-h-[400px] overflow-y-auto">
              <ActionList actions={actions} />
            </div>
          </section>
        </div>
      </div>

      {!connected && (
        <div className="fixed inset-0 bg-black/70 flex items-center justify-center">
          <div className="bg-gray-800 rounded-lg p-6 text-center">
            <div className="w-8 h-8 border-4 border-blue-500 border-t-transparent rounded-full animate-spin mx-auto mb-4" />
            <p className="text-white">Connecting to server...</p>
            <p className="text-sm text-gray-400 mt-2">Make sure the orchestrator is running</p>
          </div>
        </div>
      )}
    </div>
  )
}
