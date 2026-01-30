'use client'

import { Incident } from '@/lib/useStream'

interface Props {
  incidents: Incident[]
}

function getSeverityClass(severity: string): string {
  switch (severity) {
    case 'INCIDENT_SEVERITY_INFO':
      return 'severity-info'
    case 'INCIDENT_SEVERITY_WARNING':
      return 'severity-warning'
    case 'INCIDENT_SEVERITY_CRITICAL':
      return 'severity-critical'
    case 'INCIDENT_SEVERITY_FATAL':
      return 'severity-fatal'
    default:
      return 'bg-gray-500/20 text-gray-400 border-gray-500/50'
  }
}

function formatSeverity(severity: string): string {
  return severity.replace('INCIDENT_SEVERITY_', '').toLowerCase()
}

function formatTime(unixMs: string): string {
  return new Date(parseInt(unixMs)).toLocaleTimeString()
}

export function IncidentList({ incidents }: Props) {
  if (incidents.length === 0) {
    return (
      <div className="text-center text-gray-500 py-8">
        No incidents detected
      </div>
    )
  }

  return (
    <div className="space-y-2">
      {incidents.map((incident) => (
        <div
          key={incident.id.value}
          className={`rounded-lg p-3 border ${getSeverityClass(incident.severity)}`}
        >
          <div className="flex items-start justify-between">
            <div>
              <div className="flex items-center gap-2">
                <span className="text-xs uppercase font-semibold">
                  {formatSeverity(incident.severity)}
                </span>
                <span className="text-xs opacity-70">
                  {formatTime(incident.detectedAt.wallTimeUnixMs)}
                </span>
              </div>
              <h4 className="font-medium mt-1">{incident.title}</h4>
              <p className="text-sm opacity-80 mt-1">{incident.description}</p>
            </div>
            {incident.resolved && (
              <span className="text-xs bg-green-500/20 text-green-400 px-2 py-1 rounded">
                resolved
              </span>
            )}
          </div>
        </div>
      ))}
    </div>
  )
}
