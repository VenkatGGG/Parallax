package engine

import (
	"math/rand"
	"sync"
	"time"

	commonv1 "github.com/microcloud/gen/go/common/v1"
	simv1 "github.com/microcloud/gen/go/sim/v1"
)

// State holds the simulation ground truth
type State struct {
	mu sync.RWMutex

	nodes    map[string]*simv1.Node
	services map[string]*simv1.Service

	tickID        int64
	simTimeUnixMs int64
	startWallTime time.Time
	speedMult     float64
	simState      commonv1.SimulationState
	scenario      string
}

// NewState creates a new simulation state with default nodes and services
func NewState() *State {
	s := &State{
		nodes:         make(map[string]*simv1.Node),
		services:      make(map[string]*simv1.Service),
		tickID:        0,
		simTimeUnixMs: time.Now().UnixMilli(),
		startWallTime: time.Now(),
		speedMult:     1.0,
		simState:      commonv1.SimulationState_SIMULATION_STATE_STOPPED,
		scenario:      "normal",
	}
	s.initializeDefaultState()
	return s
}

func (s *State) initializeDefaultState() {
	zones := []string{"us-east-1a", "us-east-1b", "us-west-2a"}

	for i := 0; i < 6; i++ {
		nodeID := randomUUID()
		node := &simv1.Node{
			Id:                 &commonv1.UUID{Value: nodeID},
			Name:               nodeNames[i%len(nodeNames)],
			Status:             commonv1.NodeStatus_NODE_STATUS_HEALTHY,
			CpuUsagePercent:    rand.Float64() * 30,
			MemoryUsagePercent: rand.Float64() * 40,
			DiskUsagePercent:   rand.Float64() * 20,
			RunningServices:    int32(rand.Intn(3) + 1),
			AvailabilityZone:   zones[i%len(zones)],
			Labels:             map[string]string{"tier": "compute"},
		}
		s.nodes[nodeID] = node

		for j := 0; j < int(node.RunningServices); j++ {
			svcID := randomUUID()
			svc := &simv1.Service{
				Id:               &commonv1.UUID{Value: svcID},
				Name:             serviceNames[(i+j)%len(serviceNames)],
				NodeId:           &commonv1.UUID{Value: nodeID},
				Health:           commonv1.ServiceHealth_SERVICE_HEALTH_HEALTHY,
				RequestsPerSecond: rand.Float64() * 500,
				ErrorRatePercent:  rand.Float64() * 0.5,
				LatencyP50Ms:      rand.Float64()*10 + 5,
				LatencyP99Ms:      rand.Float64()*50 + 20,
				ReplicaCount:      int32(rand.Intn(3) + 1),
				DesiredReplicas:   3,
			}
			s.services[svcID] = svc
		}
	}
}

var nodeNames = []string{"node-alpha", "node-beta", "node-gamma", "node-delta", "node-epsilon", "node-zeta"}
var serviceNames = []string{"api-gateway", "user-service", "order-service", "payment-service", "inventory-service", "notification-service", "analytics-service", "search-service"}

func randomUUID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return string([]byte{
		hexChar(b[0]>>4), hexChar(b[0]&0xf), hexChar(b[1]>>4), hexChar(b[1]&0xf),
		hexChar(b[2]>>4), hexChar(b[2]&0xf), hexChar(b[3]>>4), hexChar(b[3]&0xf), '-',
		hexChar(b[4]>>4), hexChar(b[4]&0xf), hexChar(b[5]>>4), hexChar(b[5]&0xf), '-',
		hexChar(b[6]>>4), hexChar(b[6]&0xf), hexChar(b[7]>>4), hexChar(b[7]&0xf), '-',
		hexChar(b[8]>>4), hexChar(b[8]&0xf), hexChar(b[9]>>4), hexChar(b[9]&0xf), '-',
		hexChar(b[10]>>4), hexChar(b[10]&0xf), hexChar(b[11]>>4), hexChar(b[11]&0xf),
		hexChar(b[12]>>4), hexChar(b[12]&0xf), hexChar(b[13]>>4), hexChar(b[13]&0xf),
		hexChar(b[14]>>4), hexChar(b[14]&0xf), hexChar(b[15]>>4), hexChar(b[15]&0xf),
	})
}

func hexChar(b byte) byte {
	if b < 10 {
		return '0' + b
	}
	return 'a' + b - 10
}

// GetSimState returns the current simulation state
func (s *State) GetSimState() commonv1.SimulationState {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.simState
}

// SetSimState sets the simulation state
func (s *State) SetSimState(state commonv1.SimulationState) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.simState = state
	if state == commonv1.SimulationState_SIMULATION_STATE_RUNNING {
		s.startWallTime = time.Now()
	}
}

// GetSpeedMultiplier returns the speed multiplier
func (s *State) GetSpeedMultiplier() float64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.speedMult
}

// SetSpeedMultiplier sets the speed multiplier
func (s *State) SetSpeedMultiplier(mult float64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if mult < 0.1 {
		mult = 0.1
	}
	if mult > 10.0 {
		mult = 10.0
	}
	s.speedMult = mult
}

// GetTickID returns the current tick ID
func (s *State) GetTickID() int64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.tickID
}

// GetScenario returns the active scenario
func (s *State) GetScenario() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.scenario
}

// SetScenario sets the active scenario
func (s *State) SetScenario(scenario string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.scenario = scenario
}

// Tick advances the simulation by one tick
func (s *State) Tick(tickDuration time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.tickID++
	s.simTimeUnixMs += int64(float64(tickDuration.Milliseconds()) * s.speedMult)

	s.updateNodes()
	s.updateServices()
}

func (s *State) updateNodes() {
	for _, node := range s.nodes {
		node.CpuUsagePercent = clamp(node.CpuUsagePercent+randDelta(5), 0, 100)
		node.MemoryUsagePercent = clamp(node.MemoryUsagePercent+randDelta(2), 0, 100)
		node.DiskUsagePercent = clamp(node.DiskUsagePercent+randDelta(0.5), 0, 100)

		if node.CpuUsagePercent > 90 || node.MemoryUsagePercent > 95 {
			node.Status = commonv1.NodeStatus_NODE_STATUS_DEGRADED
		} else if node.CpuUsagePercent > 80 || node.MemoryUsagePercent > 85 {
			node.Status = commonv1.NodeStatus_NODE_STATUS_DEGRADED
		} else {
			node.Status = commonv1.NodeStatus_NODE_STATUS_HEALTHY
		}

		if s.scenario == "high_load" {
			node.CpuUsagePercent = clamp(node.CpuUsagePercent+rand.Float64()*10, 0, 100)
		}
	}
}

func (s *State) updateServices() {
	for _, svc := range s.services {
		svc.RequestsPerSecond = clamp(svc.RequestsPerSecond+randDelta(50), 0, 10000)
		svc.ErrorRatePercent = clamp(svc.ErrorRatePercent+randDelta(0.5), 0, 100)
		svc.LatencyP50Ms = clamp(svc.LatencyP50Ms+randDelta(2), 1, 1000)
		svc.LatencyP99Ms = clamp(svc.LatencyP99Ms+randDelta(10), svc.LatencyP50Ms, 5000)

		if svc.ErrorRatePercent > 10 {
			svc.Health = commonv1.ServiceHealth_SERVICE_HEALTH_CRITICAL
		} else if svc.ErrorRatePercent > 5 {
			svc.Health = commonv1.ServiceHealth_SERVICE_HEALTH_DEGRADED
		} else {
			svc.Health = commonv1.ServiceHealth_SERVICE_HEALTH_HEALTHY
		}

		if s.scenario == "cascade_failure" && rand.Float64() < 0.05 {
			svc.ErrorRatePercent = clamp(svc.ErrorRatePercent+20, 0, 100)
		}
	}
}

func randDelta(maxDelta float64) float64 {
	return (rand.Float64() - 0.5) * 2 * maxDelta
}

func clamp(v, min, max float64) float64 {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}

// Snapshot returns the current metric snapshot
func (s *State) Snapshot() *simv1.MetricSnapshot {
	s.mu.RLock()
	defer s.mu.RUnlock()

	nodes := make([]*simv1.Node, 0, len(s.nodes))
	for _, n := range s.nodes {
		nodes = append(nodes, n)
	}

	services := make([]*simv1.Service, 0, len(s.services))
	var totalRPS, totalErrors, totalLatency float64
	for _, svc := range s.services {
		services = append(services, svc)
		totalRPS += svc.RequestsPerSecond
		totalErrors += svc.ErrorRatePercent
		totalLatency += svc.LatencyP50Ms
	}

	avgErrorRate := 0.0
	avgLatency := 0.0
	if len(services) > 0 {
		avgErrorRate = totalErrors / float64(len(services))
		avgLatency = totalLatency / float64(len(services))
	}

	return &simv1.MetricSnapshot{
		Timestamp: &commonv1.SimulationTimestamp{
			TickId:        s.tickID,
			WallTimeUnixMs: time.Now().UnixMilli(),
			SimTimeUnixMs:  s.simTimeUnixMs,
		},
		Nodes:    nodes,
		Services: services,
		Traffic: &simv1.TrafficStats{
			TotalRps:          totalRPS,
			TotalErrorRate:    avgErrorRate,
			AvgLatencyMs:      avgLatency,
			ActiveConnections: int64(rand.Intn(1000) + 500),
		},
	}
}
