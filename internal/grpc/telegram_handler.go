package grpc

import (
	"context"
	api "github.com/zen-flo/telegram-service/pkg/api/proto"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type TelegramHandler struct {
	api.UnimplementedTelegramServiceServer
}

func NewTelegramHandler() *TelegramHandler {
	return &TelegramHandler{}
}

func (h *TelegramHandler) CreateSession(
	ctx context.Context,
	req *api.CreateSessionRequest,
) (*api.CreateSessionResponse, error) {

	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func (h *TelegramHandler) DeleteSession(
	ctx context.Context,
	req *api.DeleteSessionRequest,
) (*api.DeleteSessionResponse, error) {

	return nil, status.Error(codes.Unimplemented, "not implemented")
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
