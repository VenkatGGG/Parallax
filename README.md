# Microcloud Simulator

A production-grade, event-driven distributed system that simulates a self-healing cloud environment. The simulator generates telemetry from virtual nodes and services, detects anomalies, and uses autonomous agents to propose and execute remediation actions.

## Architecture Overview

```
                                    +------------------+
                                    |   ui-dashboard   |
                                    |    (Next.js)     |
                                    +--------+---------+
                                             |
                                             | SSE / Connect-RPC
                                             v
+------------------+              +------------------+              +------------------+
|                  |   gRPC       |                  |   NATS       |                  |
|   sim-engine     +------------->|   orchestrator   |<------------>|  agent-service   |
|                  |              |      (BFF)       |              |                  |
+--------+---------+              +--------+---------+              +--------+---------+
         |                                 |                                 ^
         |                                 |                                 |
         | sim.metrics                     | ops.commands                    | ops.incidents
         | sim.events                      v                                 |
         |                        +------------------+                       |
         +----------------------->|                  +-----------------------+
                                  |   NATS JetStream |
         +----------------------->|                  +-----------------------+
         |                        +--------+---------+                       |
         |                                 |                                 |
         |                                 | sim.metrics                     |
         |                                 v                                 |
         |                        +------------------+              +--------+---------+
         |                        |                  |              |                  |
         +----------------------->| signal-service   +------------->|   TimescaleDB    |
                                  |                  |              |                  |
                                  +------------------+              +------------------+
                                           |
                                           | ops.incidents
                                           v
                                  +------------------+
                                  |   agent-service  |
                                  +------------------+
```

## Technology Stack

| Layer | Technology |
|-------|------------|
| Backend Language | Go 1.23+ |
| Frontend | Next.js 16, TypeScript, Tailwind CSS |
| Async Messaging | NATS JetStream |
| Sync RPC | Connect-RPC (gRPC-compatible) |
| Data Serialization | Protocol Buffers v3 |
| Time-Series Database | TimescaleDB (PostgreSQL) |
| Containerization | Docker, Docker Compose |

## Project Structure

```
/
├── proto/                      # Protocol Buffer definitions
│   ├── common/v1/              # Shared types and enums
│   ├── sim/v1/                 # Simulation domain (metrics, control)
│   └── ops/v1/                 # Operations domain (incidents, actions)
├── gen/                        # Generated code (gitignored)
│   ├── go/                     # Go protobuf and Connect stubs
│   └── ts/                     # TypeScript protobuf stubs
├── cmd/                        # Service entry points
│   ├── sim-engine/             # Simulation engine
│   ├── signal-service/         # Metrics ingestion and detection
│   ├── agent-service/          # Incident response automation
│   └── orchestrator/           # API gateway (BFF)
├── pkg/                        # Shared libraries
│   ├── bus/                    # NATS JetStream wrapper
│   ├── storage/                # TimescaleDB repositories
│   └── logger/                 # Structured logging (slog)
├── ui-dashboard/               # Next.js frontend application
├── deployments/                # Docker Compose configuration
├── buf.yaml                    # Buf configuration
├── buf.gen.yaml                # Code generation configuration
└── go.work                     # Go workspace file
```

## Services

### sim-engine

The simulation engine maintains ground truth state for the virtual cloud environment.

**Responsibilities:**
- Manages virtual nodes and services with realistic resource metrics
- Runs a tick-based simulation loop with configurable speed
- Publishes metric snapshots to NATS at high frequency
- Exposes gRPC control API for play/pause/speed adjustment
- Supports multiple scenarios: normal, high_load, cascade_failure

**NATS Subjects:**
- Publishes: `sim.metrics`, `sim.events`
- Subscribes: `ops.commands`

### signal-service

The signal service acts as the observability layer, ingesting metrics and detecting anomalies.

**Responsibilities:**
- Consumes metrics from NATS JetStream
- Persists time-series data to TimescaleDB hypertables
- Applies sliding window detection rules
- Publishes incidents when thresholds are breached

**Detection Rules:**
- High/critical error rate (>5%, >10%)
- High/critical CPU usage (>85%, >95%)
- High memory usage (>90%)
- High P99 latency (>500ms)

**NATS Subjects:**
- Subscribes: `sim.metrics`
- Publishes: `ops.incidents`

### agent-service

The agent service implements autonomous remediation through rule-based decision logic.

**Responsibilities:**
- Listens for incidents from signal-service
- Applies decision rules to determine appropriate actions
- Proposes remediation actions with tick-based idempotency
- Implements cooldown periods to prevent action storms

**Action Types:**
- `RESTART_SERVICE`: Restart a failing service
- `SCALE_UP`: Increase replica count
- `SCALE_DOWN`: Decrease replica count
- `DRAIN_NODE`: Take a node offline
- `REBALANCE_TRAFFIC`: Redistribute load

**NATS Subjects:**
- Subscribes: `ops.incidents`
- Publishes: `ops.actions`

### orchestrator

The orchestrator serves as the Backend-for-Frontend (BFF) gateway.

**Responsibilities:**
- Exposes Connect-RPC APIs for action management
- Provides SSE streaming endpoint for real-time UI updates
- Manages action approval/rejection workflow
- Publishes approved actions as commands to sim-engine

**API Endpoints:**
- `POST /ops.v1.ActionService/ListPendingActions`
- `POST /ops.v1.ActionService/ApproveAction`
- `POST /ops.v1.ActionService/RejectAction`
- `POST /ops.v1.ActionService/GetActionHistory`
- `GET /api/stream` (SSE)
- `GET /health`

### ui-dashboard

The dashboard provides real-time visualization of the simulation state.

**Features:**
- Real-time metrics display via SSE
- Node and service health visualization
- Incident feed with severity indicators
- Action approval/rejection interface
- Traffic statistics overview

## Getting Started

### Prerequisites

- Go 1.23 or later
- Node.js 20 or later
- Docker and Docker Compose
- Buf CLI (`brew install bufbuild/buf/buf`)

### Infrastructure Setup

Start NATS and TimescaleDB:

```bash
cd deployments
docker-compose up -d
```

### Generate Protocol Buffer Code

```bash
buf generate
```

### Run Services

Start each service in a separate terminal:

```bash
# Terminal 1: Simulation Engine
go run ./cmd/sim-engine

# Terminal 2: Signal Service
go run ./cmd/signal-service

# Terminal 3: Agent Service
go run ./cmd/agent-service

# Terminal 4: Orchestrator
go run ./cmd/orchestrator
```

### Run Dashboard

```bash
cd ui-dashboard
npm install
npm run dev
```

Open http://localhost:3000 in your browser.

## Configuration

All services follow 12-factor app principles and are configured via environment variables.

### sim-engine

| Variable | Default | Description |
|----------|---------|-------------|
| `NATS_URL` | `nats://localhost:4222` | NATS server URL |
| `ADDR` | `:8080` | gRPC server address |
| `LOG_LEVEL` | `info` | Logging level |
| `LOG_FORMAT` | `json` | Log format (json/text) |

### signal-service

| Variable | Default | Description |
|----------|---------|-------------|
| `NATS_URL` | `nats://localhost:4222` | NATS server URL |
| `DB_HOST` | `localhost` | TimescaleDB host |
| `DB_PORT` | `5432` | TimescaleDB port |
| `DB_NAME` | `microcloud` | Database name |
| `DB_USER` | `microcloud` | Database user |
| `DB_PASSWORD` | `microcloud` | Database password |

### agent-service

| Variable | Default | Description |
|----------|---------|-------------|
| `NATS_URL` | `nats://localhost:4222` | NATS server URL |
| `DB_HOST` | `localhost` | TimescaleDB host |
| `DB_PORT` | `5432` | TimescaleDB port |

### orchestrator

| Variable | Default | Description |
|----------|---------|-------------|
| `NATS_URL` | `nats://localhost:4222` | NATS server URL |
| `DB_HOST` | `localhost` | TimescaleDB host |
| `ADDR` | `:8081` | HTTP server address |

### ui-dashboard

| Variable | Default | Description |
|----------|---------|-------------|
| `NEXT_PUBLIC_API_URL` | `http://localhost:8081` | Orchestrator API URL |
| `NEXT_PUBLIC_STREAM_URL` | `http://localhost:8081/api/stream` | SSE stream URL |

## NATS Subject Schema

| Subject | Publisher | Subscriber | Payload |
|---------|-----------|------------|---------|
| `sim.metrics` | sim-engine | signal-service, orchestrator | `MetricSnapshot` |
| `sim.events` | sim-engine | orchestrator | `SimulationEvent` |
| `ops.incidents` | signal-service | agent-service, orchestrator | `Incident` |
| `ops.actions` | agent-service | orchestrator | `Action` |
| `ops.commands` | orchestrator | sim-engine | `ApplyActionCommand` |

## Database Schema

### metrics (TimescaleDB Hypertable)

| Column | Type | Description |
|--------|------|-------------|
| `time` | `TIMESTAMPTZ` | Metric timestamp |
| `tick_id` | `BIGINT` | Simulation tick ID |
| `node_id` | `TEXT` | Node identifier |
| `service_id` | `TEXT` | Service identifier |
| `metric_name` | `TEXT` | Metric name |
| `metric_value` | `DOUBLE PRECISION` | Metric value |
| `labels` | `JSONB` | Additional labels |

### incidents

| Column | Type | Description |
|--------|------|-------------|
| `id` | `UUID` | Primary key |
| `detected_at` | `TIMESTAMPTZ` | Detection timestamp |
| `tick_id` | `BIGINT` | Detection tick |
| `severity` | `INT` | Severity level (1-4) |
| `title` | `TEXT` | Incident title |
| `description` | `TEXT` | Detailed description |
| `rule_name` | `TEXT` | Triggering rule |
| `resolved` | `BOOLEAN` | Resolution status |

### actions

| Column | Type | Description |
|--------|------|-------------|
| `id` | `UUID` | Primary key |
| `incident_id` | `UUID` | Related incident |
| `proposed_at_tick` | `BIGINT` | Proposal tick (for idempotency) |
| `action_type` | `INT` | Action type enum |
| `target_id` | `TEXT` | Target node/service |
| `status` | `INT` | Action status enum |
| `reason` | `TEXT` | Justification |

## Development

### Adding New Detection Rules

Edit `cmd/signal-service/detector/rules.go`:

```go
Rule{
    Name:          "custom_rule",
    MetricName:    "metric_name",
    Operator:      "gt",  // gt, gte, lt, lte, eq
    Threshold:     50.0,
    WindowSeconds: 30,
    Severity:      commonv1.IncidentSeverity_INCIDENT_SEVERITY_WARNING,
}
```

### Adding New Action Types

1. Add enum value in `proto/common/v1/enums.proto`
2. Regenerate code: `buf generate`
3. Add handling in `cmd/agent-service/decider/decider.go`
4. Add execution logic in `cmd/sim-engine/engine/engine.go`

### Running Tests

```bash
go test ./pkg/...
go test ./cmd/...
```

## License

Proprietary - All rights reserved.
