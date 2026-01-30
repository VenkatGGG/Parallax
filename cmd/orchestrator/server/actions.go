package server

import (
	"context"
	"log/slog"

	"connectrpc.com/connect"

	"github.com/microcloud/bus"
	commonv1 "github.com/microcloud/gen/go/common/v1"
	opsv1 "github.com/microcloud/gen/go/ops/v1"
	"github.com/microcloud/gen/go/ops/v1/opsv1connect"
	"github.com/microcloud/storage"
)

// ActionServer implements the ActionService
type ActionServer struct {
	actionsRepo *storage.ActionsRepository
	publisher   *bus.Publisher
	log         *slog.Logger
}

var _ opsv1connect.ActionServiceHandler = (*ActionServer)(nil)

// NewActionServer creates a new action server
func NewActionServer(actionsRepo *storage.ActionsRepository, publisher *bus.Publisher, log *slog.Logger) *ActionServer {
	return &ActionServer{
		actionsRepo: actionsRepo,
		publisher:   publisher,
		log:         log,
	}
}

// ListPendingActions returns all pending actions
func (s *ActionServer) ListPendingActions(ctx context.Context, req *connect.Request[opsv1.ListPendingActionsRequest]) (*connect.Response[opsv1.ListPendingActionsResponse], error) {
	limit := int(req.Msg.Limit)
	if limit <= 0 {
		limit = 50
	}

	rows, err := s.actionsRepo.ListPending(ctx, limit)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	actions := make([]*opsv1.Action, 0, len(rows))
	for _, row := range rows {
		actions = append(actions, rowToAction(row))
	}

	return connect.NewResponse(&opsv1.ListPendingActionsResponse{
		Actions: actions,
	}), nil
}

// ApproveAction approves a pending action
func (s *ActionServer) ApproveAction(ctx context.Context, req *connect.Request[opsv1.ApproveActionRequest]) (*connect.Response[opsv1.ApproveActionResponse], error) {
	actionID := req.Msg.ActionId.Value

	action, err := s.actionsRepo.GetByID(ctx, actionID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if action == nil {
		return nil, connect.NewError(connect.CodeNotFound, nil)
	}

	if err := s.actionsRepo.Approve(ctx, actionID); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	cmd := &opsv1.ApplyActionCommand{
		ActionId:     &commonv1.UUID{Value: actionID},
		TargetTickId: action.ProposedAtTick,
		ActionType:   commonv1.ActionType(action.ActionType),
		TargetId:     action.TargetID,
		Parameters:   action.Parameters,
	}

	if err := s.publisher.PublishCommand(ctx, cmd); err != nil {
		s.log.Error("failed to publish command", "error", err)
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	s.log.Info("action approved", "action_id", actionID)

	return connect.NewResponse(&opsv1.ApproveActionResponse{
		Success: true,
		Message: "Action approved and command published",
	}), nil
}

// RejectAction rejects a pending action
func (s *ActionServer) RejectAction(ctx context.Context, req *connect.Request[opsv1.RejectActionRequest]) (*connect.Response[opsv1.RejectActionResponse], error) {
	actionID := req.Msg.ActionId.Value
	reason := req.Msg.Reason

	if err := s.actionsRepo.Reject(ctx, actionID, reason); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	s.log.Info("action rejected", "action_id", actionID, "reason", reason)

	return connect.NewResponse(&opsv1.RejectActionResponse{
		Success: true,
	}), nil
}

// GetActionHistory returns recent actions
func (s *ActionServer) GetActionHistory(ctx context.Context, req *connect.Request[opsv1.GetActionHistoryRequest]) (*connect.Response[opsv1.GetActionHistoryResponse], error) {
	limit := int(req.Msg.Limit)
	if limit <= 0 {
		limit = 100
	}

	rows, err := s.actionsRepo.ListRecent(ctx, limit)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	actions := make([]*opsv1.Action, 0, len(rows))
	for _, row := range rows {
		actions = append(actions, rowToAction(row))
	}

	return connect.NewResponse(&opsv1.GetActionHistoryResponse{
		Actions:    actions,
		TotalCount: int32(len(actions)),
	}), nil
}

func rowToAction(row storage.ActionRow) *opsv1.Action {
	action := &opsv1.Action{
		Id:             &commonv1.UUID{Value: row.ID},
		IncidentId:     &commonv1.UUID{Value: row.IncidentID},
		ProposedAtTick: row.ProposedAtTick,
		ActionType:     commonv1.ActionType(row.ActionType),
		TargetId:       row.TargetID,
		Status:         commonv1.ActionStatus(row.Status),
		Reason:         row.Reason,
		Parameters:     row.Parameters,
		CreatedAt: &commonv1.SimulationTimestamp{
			WallTimeUnixMs: row.CreatedAt.UnixMilli(),
		},
		ResultMessage: row.ResultMessage,
	}

	if row.ExecutedAt != nil {
		action.ExecutedAt = &commonv1.SimulationTimestamp{
			WallTimeUnixMs: row.ExecutedAt.UnixMilli(),
		}
	}

	return action
}
