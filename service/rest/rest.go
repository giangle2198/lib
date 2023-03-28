package rest

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/rs/cors"
	"go.uber.org/zap"
)

type RestfulService struct {
	address int
	statik  http.FileSystem
	cors    bool
	server  *http.Server
	mux     *runtime.ServeMux
}

func NewRestfulService(options ...func(*RestfulService)) *RestfulService {
	svr := &RestfulService{}
	for _, o := range options {
		o(svr)
	}
	return svr
}

func WithStatik(statik http.FileSystem) func(*RestfulService) {
	return func(r *RestfulService) {
		r.statik = statik
	}
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
	if err := r.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		zap.S().Panic("Gateway: Failed to listen and serve", zap.Error(err))
	}
}

func (r *RestfulService) Close() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := r.server.Shutdown(ctx); err != nil {
		zap.S().Fatalw("Failed to shutdown gateway", zap.Error(err))
	}
}

func (r *RestfulService) Init() {
	mux := runtime.NewServeMux(runtime.WithMarshalerOption(runtime.MIMEWildcard, &runtime.JSONPb{}))

	httpMux := http.NewServeMux()
	httpMux.Handle("/bim/swagger-ui/", http.StripPrefix("/bim/swagger-ui/", http.FileServer(r.statik)))
	httpMux.Handle("/", mux)

	var httpHandler http.Handler = httpMux
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
		httpHandler = corsHandler.Handler(httpMux)
	}

	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", r.address),
		Handler: httpHandler,
	}

	r.server = server
}
