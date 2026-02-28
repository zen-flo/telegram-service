package grpc

import (
	"context"
	"fmt"
	"net"

	api "github.com/zen-flo/telegram-service/pkg/api/proto"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

type Server struct {
	grpcServer *grpc.Server
	logger     *zap.Logger
	port       int
}

func NewServer(
	port int,
	logger *zap.Logger,
	telegramHandler api.TelegramServiceServer,
) *Server {

	grpcServer := grpc.NewServer()

	api.RegisterTelegramServiceServer(grpcServer, telegramHandler)

	// Enable reflection for grpcurl
	reflection.Register(grpcServer)

	return &Server{
		grpcServer: grpcServer,
		logger:     logger,
		port:       port,
	}
}

func (s *Server) Start(ctx context.Context) error {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", s.port))
	if err != nil {
		return fmt.Errorf("failed to listen: %w", err)
	}

	go func() {
		<-ctx.Done()
		s.logger.Info("shutting down grpc server")
		s.grpcServer.GracefulStop()
	}()

	s.logger.Info("grpc server started", zap.Int("port", s.port))
	return s.grpcServer.Serve(lis)
}
