package micro

import (
	"context"
	"fmt"
	"lib/log"
	"net"
	"strings"

	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_zap "github.com/grpc-ecosystem/go-grpc-middleware/logging/zap"
	grpc_opentracing "github.com/grpc-ecosystem/go-grpc-middleware/tracing/opentracing"
	grpc_validator "github.com/grpc-ecosystem/go-grpc-middleware/validator"
	grpc_prometheus "github.com/grpc-ecosystem/go-grpc-prometheus"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type MicroService struct {
	address     int
	middleware  bool
	server      *grpc.Server
	interceptor grpc.ServerOption
}

func NewMicroService(options ...func(*MicroService)) *MicroService {
	svr := &MicroService{}
	for _, o := range options {
		o(svr)
	}
	return svr
}

func (m *MicroService) Run() {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", m.address))
	if err != nil {
		zap.S().Panic("Server: Can't create listener", zap.Error(err))
	}

	if err := m.server.Serve(listener); err != nil {
		zap.S().Panic("Server: Failed to serve", zap.Error(err))
	}
}

func (m *MicroService) Close() {
	m.server.GracefulStop()
}

func (m *MicroService) Setup() *grpc.Server {
	if m.middleware {
		m.server = grpc.NewServer(m.interceptor)
	} else {
		m.server = grpc.NewServer()
	}

	return m.server
}

func (m *MicroService) GetService() *grpc.Server {
	return m.server
}

func WithMiddleware(interceptors ...grpc.UnaryServerInterceptor) func(*MicroService) {
	return func(m *MicroService) {
		zapFunc := func(ctx context.Context, fullMethodName string, servingObject interface{}) bool {
			return true
		}
		opentracingFunc := func(ctx context.Context, fullMethodName string) bool {
			if strings.Contains(fullMethodName, "Liveness") || strings.Contains(fullMethodName, "Readiness") {
				return false
			}
			return true
		}

		m.interceptor = grpc_middleware.WithUnaryServerChain(
			grpc_middleware.ChainUnaryServer(interceptors...),
			grpc_validator.UnaryServerInterceptor(),
			unpanicGRPC(),
			grpc_validator.UnaryServerInterceptor(),
			grpc_opentracing.UnaryServerInterceptor([]grpc_opentracing.Option{
				grpc_opentracing.WithFilterFunc(opentracingFunc),
			}...),
			grpc_prometheus.UnaryServerInterceptor,
			grpc_zap.PayloadUnaryServerInterceptor(zap.S().Desugar(), zapFunc),
		)

		m.middleware = true
	}
}

func WithAddress(address int) func(*MicroService) {
	return func(ms *MicroService) {
		ms.address = address
	}
}

// Unpanic for GRPC
func unpanicGRPC() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
		defer func() {
			if errInf := recover(); errInf != nil {
				log.Logger.Error("Recover from unpanic grpc:%v, %+v", errInf, resp)
				// _ = handleResponse(resp)
				err = status.Errorf(codes.Internal, "Panic from server")
			}
		}()

		return handler(ctx, req)
	}
}
