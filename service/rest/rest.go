package rest

import (
	"context"
	"fmt"
	"net/http"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/rs/cors"
	"go.uber.org/zap"
)

type RestfulService struct {
	address int
	statik  http.FileSystem
	cors    bool
	server  *http.Server
	httpMux *http.ServeMux
	mux     *runtime.ServeMux
}

func NewRestfulService(options ...func(*RestfulService)) *RestfulService {
	mux := runtime.NewServeMux(runtime.WithMarshalerOption(runtime.MIMEWildcard, &runtime.JSONPb{}))

	svr := &RestfulService{
		mux: mux,
	}
	for _, o := range options {
		o(svr)
	}

	return svr
}

func WithCors(cors bool) func(*RestfulService) {
	return func(r *RestfulService) {
		r.cors = cors
	}
}

func WithAddress(address int) func(*RestfulService) {
	return func(r *RestfulService) {
		r.address = address
	}
}

func (r *RestfulService) GetService() *http.Server {
	return r.server
}

func (r *RestfulService) GetMux() *runtime.ServeMux {
	return r.mux
}

func (r *RestfulService) Run() {
	var httpHandler http.Handler = r.httpMux
	if r.cors {
		corsHandler := cors.New(cors.Options{
			AllowedOrigins:   []string{"*"},
			AllowCredentials: true,
			AllowedMethods:   []string{"GET", "PUT", "POST", "DELETE", "PATCH", "OPTIONS"},
			AllowedHeaders:   []string{"*"},
			ExposedHeaders: []string{"Grpc-Metadata-Authorization",
				"Content-Type", "Content-Disposition",
				"File-Name",
				"Content-Transfer-Encoding",
				"Grpc-Metadata-Custom-Header-Additional-Info"},
			Debug: true,
		})
		httpHandler = corsHandler.Handler(r.httpMux)
	}

	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", r.address),
		Handler: httpHandler,
	}

	r.server = server

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		zap.S().Panic("Gateway: Failed to listen and serve", zap.Error(err))
	}
}

func (r *RestfulService) Close() {
	// ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	// defer cancel()
	if err := r.server.Shutdown(context.Background()); err != nil {
		zap.S().Fatalw("Failed to shutdown gateway", zap.Error(err))
	}
}

func (r *RestfulService) Handle(pattern string, handler http.Handler) {
	r.httpMux.Handle(pattern, handler)
}

func (r *RestfulService) Setup() *http.ServeMux {

	httpMux := http.NewServeMux()
	httpMux.Handle("/", r.mux)
	r.httpMux = httpMux

	return httpMux
}
