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

	qrURL, err := s.StartQR(func() {
		s.MarkReady()
		h.logger.Info("session authorized", zap.String("session_id", s.ID()))
	})
	if err != nil {
		h.logger.Error("failed to start qr auth",
			zap.Error(err),
		)
		return nil, status.Error(codes.Internal, "failed to start qr auth")
	}

	h.logger.Info("session created",
		zap.String("session_id", s.ID()),
	)

	return &api.CreateSessionResponse{
		SessionId: stringPtr(s.ID()),
		QrCode:    stringPtr(qrURL),
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

	s, err := h.manager.Get(req.GetSessionId())
	if err != nil {
		return nil, status.Error(codes.NotFound, "session not found")
	}

	if !s.IsReady() {
		return nil, status.Error(codes.FailedPrecondition, "session not authorized")
	}

	msgID, err := s.SendMessage(
		req.GetPeer(),
		req.GetText(),
	)

	if err != nil {
		h.logger.Error("failed to send message", zap.Error(err))
		return nil, status.Error(codes.Internal, "failed to send message")
	}

	return &api.SendMessageResponse{
		MessageId: int64Ptr(msgID),
	}, nil
}

func (h *TelegramHandler) SubscribeMessages(
	req *api.SubscribeMessagesRequest,
	stream api.TelegramService_SubscribeMessagesServer,
) error {

	return status.Error(codes.Unimplemented, "not implemented")
}

func (h *TelegramHandler) GetSessionStatus(
	ctx context.Context,
	req *api.GetSessionStatusRequest,
) (*api.GetSessionStatusResponse, error) {

	s, err := h.manager.Get(req.GetSessionId())
	if err != nil {
		if errors.Is(err, session.ErrSessionNotFound) {
			return nil, status.Error(codes.NotFound, "session not found")
		}
		return nil, status.Error(codes.Internal, "internal error")
	}

	return &api.GetSessionStatusResponse{
		Ready: boolPtr(s.IsReady()),
	}, nil
}

func stringPtr(s string) *string {
	return &s
}

func int64Ptr(i int64) *int64 {
	return &i
}

func boolPtr(b bool) *bool {
	return &b
}
