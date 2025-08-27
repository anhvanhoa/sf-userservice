package grpcservice

import (
	"context"
	"fmt"
	"net"

	"auth-service/bootstrap"
	loggerI "auth-service/domain/service/logger"

	"buf.build/go/protovalidate"
	proto_auth "github.com/anhvanhoa/sf-proto/gen/auth/v1"
	protovalidate_middleware "github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/protovalidate"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
)

type GRPCServer struct {
	server    *grpc.Server
	healthSrv *health.Server
	env       *bootstrap.Env
	log       loggerI.Log
}

func NewGRPCServer(env *bootstrap.Env, authService proto_auth.AuthServiceServer, log loggerI.Log) *GRPCServer {
	validator, err := protovalidate.New()
	if err != nil {
		panic(err)
	}
	server := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			LoggingInterceptor(log),
			protovalidate_middleware.UnaryServerInterceptor(validator),
		),
	)

	proto_auth.RegisterAuthServiceServer(server, authService)

	healthSrv := health.NewServer()
	grpc_health_v1.RegisterHealthServer(server, healthSrv)

	healthSrv.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)
	healthSrv.SetServingStatus(env.NAME_SERVICE, grpc_health_v1.HealthCheckResponse_SERVING)

	if !env.IsProduction() {
		log.Info("Reflection is enabled")
		reflection.Register(server)
	}

	return &GRPCServer{
		server:    server,
		healthSrv: healthSrv,
		log:       log,
		env:       env,
	}
}

func (s *GRPCServer) Start(ctx context.Context) error {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", s.env.PORT_GRPC))
	if err != nil {
		return fmt.Errorf("failed to listen: %v", err)
	}

	go func() {
		s.log.Info(fmt.Sprintf("gRPC server starting to serve on port %d", s.env.PORT_GRPC))
		if err := s.server.Serve(lis); err != nil {
			s.log.Error(fmt.Sprintf("gRPC server failed to serve: %v", err))
		}
	}()

	<-ctx.Done()

	s.log.Info("Shutting down gRPC server...")
	s.server.GracefulStop()

	return nil
}

func (s *GRPCServer) Stop() {
	s.server.GracefulStop()
}

func (s *GRPCServer) GetServer() *grpc.Server {
	return s.server
}

func (s *GRPCServer) SetHealthStatus(service string, status grpc_health_v1.HealthCheckResponse_ServingStatus) {
	if s.healthSrv != nil {
		s.healthSrv.SetServingStatus(service, status)
	}
}
