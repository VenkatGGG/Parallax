package server

import (
	"context"
	"log/slog"

	"connectrpc.com/connect"

	simv1 "github.com/microcloud/gen/go/sim/v1"
	"github.com/microcloud/gen/go/sim/v1/simv1connect"
	"github.com/microcloud/sim-engine/engine"
)

// ControlServer implements the SimulationControl service
type ControlServer struct {
	engine *engine.Engine
	log    *slog.Logger
}

var _ simv1connect.SimulationControlHandler = (*ControlServer)(nil)

// NewControlServer creates a new control server
func NewControlServer(eng *engine.Engine, log *slog.Logger) *ControlServer {
	return &ControlServer{
		engine: eng,
		log:    log,
	}
}

// GetState returns the current simulation state
func (s *ControlServer) GetState(ctx context.Context, req *connect.Request[simv1.GetStateRequest]) (*connect.Response[simv1.GetStateResponse], error) {
	state := s.engine.State()
	return connect.NewResponse(&simv1.GetStateResponse{
		State:           state.GetSimState(),
		SpeedMultiplier: state.GetSpeedMultiplier(),
		CurrentTick:     state.GetTickID(),
		ActiveScenario:  state.GetScenario(),
	}), nil
}

// SetState sets the simulation state (play/pause/stop)
func (s *ControlServer) SetState(ctx context.Context, req *connect.Request[simv1.SetStateRequest]) (*connect.Response[simv1.SetStateResponse], error) {
	state := s.engine.State()
	newState := req.Msg.State

	oldState := state.GetSimState()
	state.SetSimState(newState)

	s.log.Info("simulation state changed", "from", oldState, "to", newState)

	return connect.NewResponse(&simv1.SetStateResponse{
		State: newState,
	}), nil
}

// SetSpeed sets the simulation speed multiplier
func (s *ControlServer) SetSpeed(ctx context.Context, req *connect.Request[simv1.SetSpeedRequest]) (*connect.Response[simv1.SetSpeedResponse], error) {
	state := s.engine.State()
	state.SetSpeedMultiplier(req.Msg.SpeedMultiplier)

	s.log.Info("simulation speed changed", "multiplier", req.Msg.SpeedMultiplier)

	return connect.NewResponse(&simv1.SetSpeedResponse{
		SpeedMultiplier: state.GetSpeedMultiplier(),
	}), nil
}

// LoadScenario loads a simulation scenario
func (s *ControlServer) LoadScenario(ctx context.Context, req *connect.Request[simv1.LoadScenarioRequest]) (*connect.Response[simv1.LoadScenarioResponse], error) {
	state := s.engine.State()
	scenario := req.Msg.ScenarioName

	validScenarios := map[string]bool{
		"normal":          true,
		"high_load":       true,
		"cascade_failure": true,
	}

	if !validScenarios[scenario] {
		return connect.NewResponse(&simv1.LoadScenarioResponse{
			Success: false,
			Message: "unknown scenario: " + scenario,
		}), nil
	}

	state.SetScenario(scenario)
	s.log.Info("scenario loaded", "scenario", scenario)

	return connect.NewResponse(&simv1.LoadScenarioResponse{
		Success: true,
		Message: "scenario loaded: " + scenario,
	}), nil
}
