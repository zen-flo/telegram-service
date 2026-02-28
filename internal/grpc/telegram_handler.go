package grpc

import (
	"context"
	"errors"
	"github.com/zen-flo/telegram-service/internal/session"
	api "github.com/zen-flo/telegram-service/pkg/api/proto"
	"go.uber.org/zap"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type TelegramHandler struct {
	api.UnimplementedTelegramServiceServer

	manager *session.Manager
	logger  *zap.Logger
}

func NewTelegramHandler(
	manager *session.Manager,
	logger *zap.Logger,
) *TelegramHandler {
	return &TelegramHandler{
		manager: manager,
		logger:  logger,
	}
}

func (h *TelegramHandler) CreateSession(
	ctx context.Context,
	req *api.CreateSessionRequest,
) (*api.CreateSessionResponse, error) {

	s, err := h.manager.Create()
	if err != nil {
		h.logger.Error("failed to create session", zap.Error(err))
		return nil, status.Error(codes.Internal, "failed to create session")
	}

	h.logger.Info("session created",
		zap.String("session_id", s.ID()),
	)

	return &api.CreateSessionResponse{
		SessionId: stringPtr(s.ID()),
	}, nil
}

func (h *TelegramHandler) DeleteSession(
	ctx context.Context,
	req *api.DeleteSessionRequest,
) (*api.DeleteSessionResponse, error) {

	err := h.manager.Delete(req.GetSessionId())
	if err != nil {
		if errors.Is(err, session.ErrSessionNotFound) {
			return nil, status.Error(codes.NotFound, "session not found")
		}

		h.logger.Error(
			"failed to delete session",
			zap.Error(err),
		)
		return nil, status.Error(codes.Internal, "failed to delete session")
	}

	h.logger.Info(
		"session deleted",
		zap.String("session_id", req.GetSessionId()),
	)

	return &api.DeleteSessionResponse{}, nil
}

func (h *TelegramHandler) SendMessage(
	ctx context.Context,
	req *api.SendMessageRequest,
) (*api.SendMessageResponse, error) {

	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func (h *TelegramHandler) SubscribeMessages(
	req *api.SubscribeMessagesRequest,
	stream api.TelegramService_SubscribeMessagesServer,
) error {

	return status.Error(codes.Unimplemented, "not implemented")
}

func stringPtr(s string) *string {
	return &s
}
