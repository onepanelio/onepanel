package main

import (
	"context"
	"flag"
	"net"
	"net/http"
	"os"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/gorilla/handlers"
	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_recovery "github.com/grpc-ecosystem/go-grpc-middleware/recovery"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/onepanelio/core/api"
	"github.com/onepanelio/core/kube"
	"github.com/onepanelio/core/manager"
	"github.com/onepanelio/core/repository"
	"github.com/onepanelio/core/server"
	"github.com/pressly/goose"
	log "github.com/sirupsen/logrus"
	"github.com/tmc/grpc-websocket-proxy/wsproxy"
	"google.golang.org/grpc"
)

var (
	rpcPort      = flag.String("rpc-port", ":8887", "RPC Port")
	httpPort     = flag.String("http-port", ":8888", "RPC Port")
	recoveryFunc grpc_recovery.RecoveryHandlerFunc
)

func main() {
	flag.Parse()

	db := repository.NewDB(os.Getenv("DB_DRIVER_NAME"), os.Getenv("DB_DATASOURCE_NAME"))
	if err := goose.Run("up", db.Base(), "db"); err != nil {
		log.Fatalf("goose up: %v", err)
	}

	kubeConfig := kube.NewConfig()

	go startRPCServer(db, kubeConfig)
	startHTTPProxy()
}

func startRPCServer(db *repository.DB, kubeConfig *kube.Config) {
	resourceManager := manager.NewResourceManager(db, kubeConfig)

	log.Printf("Starting RPC server on port %v", *rpcPort)
	lis, err := net.Listen("tcp", *rpcPort)
	if err != nil {
		log.Fatalf("Failed to start RPC listener: %v", err)
	}
	recoveryFunc = func(p interface{}) (err error) {
		return status.Errorf(codes.Unknown, "panic triggered: %v", p)
	}
	opts := []grpc_recovery.Option{
		grpc_recovery.WithRecoveryHandler(recoveryFunc),
	}
	s := grpc.NewServer(grpc.UnaryInterceptor(
		grpc_middleware.ChainUnaryServer(loggingInterceptor,
			grpc_recovery.UnaryServerInterceptor(opts...))))
	api.RegisterWorkflowServiceServer(s, server.NewWorkflowServer(resourceManager))
	api.RegisterSecretServiceServer(s, server.NewSecretServer(kubeConfig))
	api.RegisterNamespaceServiceServer(s, server.NewNamespaceServer(kubeConfig))

	if err := s.Serve(lis); err != nil {
		log.Fatalf("Failed to serve RPC server: %v", err)
	}
}

func startHTTPProxy() {
	endpoint := "localhost" + *rpcPort
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Register gRPC server endpoint
	// Note: Make sure the gRPC server is running properly and accessible
	mux := runtime.NewServeMux()
	opts := []grpc.DialOption{grpc.WithInsecure()}

	registerHandler(api.RegisterWorkflowServiceHandlerFromEndpoint, ctx, mux, endpoint, opts)
	registerHandler(api.RegisterSecretServiceHandlerFromEndpoint, ctx, mux, endpoint, opts)
	registerHandler(api.RegisterNamespaceServiceHandlerFromEndpoint, ctx, mux, endpoint, opts)

	log.Printf("Starting HTTP proxy on port %v", *httpPort)

	// Allow all origins
	ogValidator := func(str string) bool {
		return true
	}

	// Allow Content-Type for JSON
	allowedHeaders := handlers.AllowedHeaders([]string{"Content-Type"})

	// Allow PUT. Have to include all others as it clears them out.
	allowedMethods := handlers.AllowedMethods([]string{"HEAD", "GET", "POST", "PUT", "DELETE", "PATCH"})

	if err := http.ListenAndServe(*httpPort, wsproxy.WebsocketProxy(handlers.CORS(handlers.AllowedOriginValidator(ogValidator), allowedHeaders, allowedMethods)(mux))); err != nil {
		log.Fatalf("Failed to serve HTTP listener: %v", err)
	}
}

type registerFunc func(ctx context.Context, mux *runtime.ServeMux, endpoint string, opts []grpc.DialOption) error

func registerHandler(register registerFunc, ctx context.Context, mux *runtime.ServeMux, endpoint string, opts []grpc.DialOption) {
	err := register(ctx, mux, endpoint, opts)
	if err != nil {
		log.Fatalf("Failed to register handler: %v", err)
	}
}

func loggingInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
	log.WithFields(log.Fields{
		"fullMethod": info.FullMethod,
	}).Info("handler started")
	resp, err = handler(ctx, req)
	if err != nil {
		log.WithFields(log.Fields{
			"fullMethod": info.FullMethod,
		}).Warning(err)
		return
	}
	log.WithFields(log.Fields{
		"fullMethod": info.FullMethod,
	}).Info("handler finished")
	return
}
