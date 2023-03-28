package micro

import (
	"context"
	"fmt"
	"net"

	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_zap "github.com/grpc-ecosystem/go-grpc-middleware/logging/zap"
	grpc_validator "github.com/grpc-ecosystem/go-grpc-middleware/validator"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

type MicroService struct {
	address     int
	middleware  bool
	server      *grpc.Server
	interceptor grpc.UnaryServerInterceptor
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

func (m *MicroService) Init() {
	zapFunc := func(ctx context.Context, fullMethodName string, servingObject interface{}) bool {
		return true
	}

	var server *grpc.Server

	if m.middleware {

		server = grpc.NewServer(
			grpc_middleware.WithUnaryServerChain(
				m.interceptor,
				grpc_validator.UnaryServerInterceptor(),
				grpc_zap.PayloadUnaryServerInterceptor(zap.S().Desugar(), zapFunc),
			),
		)

	} else {
		server = grpc.NewServer()
	}

	m.server = server
}

func (m *MicroService) GetService() *grpc.Server {
	return m.server
}

func WithMiddleware(middleware bool, interceptor grpc.UnaryServerInterceptor) func(*MicroService) {
	return func(ms *MicroService) {
		ms.middleware = middleware
		ms.interceptor = interceptor
	}
}

func WithAddress(address int) func(*MicroService) {
	return func(ms *MicroService) {
		ms.address = address
	}
}
