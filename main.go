package main

import (
	"context"
	"flag"
	"fmt"
	"net"
	"net/http"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/gorilla/handlers"
	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_logrus "github.com/grpc-ecosystem/go-grpc-middleware/logging/logrus"
	grpc_recovery "github.com/grpc-ecosystem/go-grpc-middleware/recovery"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/jmoiron/sqlx"
	"github.com/onepanelio/core/api"
	v1 "github.com/onepanelio/core/pkg"
	"github.com/onepanelio/core/pkg/util/env"
	"github.com/onepanelio/core/server"
	"github.com/onepanelio/core/server/auth"
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

	kubeConfig := v1.NewConfig()
	client, err := v1.NewClient(kubeConfig, nil)
	if err != nil {
		return
	}
	config, err := client.GetSystemConfig()
	if err != nil {
		log.Fatalf("Failed to get system config: %v", err)
	}

	databaseDataSourceName := fmt.Sprintf("host=%v user=%v password=%v dbname=%v sslmode=disable",
		config["databaseHost"], config["databaseUsername"], config["databasePassword"], config["databaseName"])
	db := sqlx.MustConnect(config["databaseDriverName"], databaseDataSourceName)
	if err := goose.Run("up", db.DB, "db"); err != nil {
		log.Fatalf("Failed to run database migrations: %v", err)
	}

	go startRPCServer(db, kubeConfig)
	startHTTPProxy()
}

func startRPCServer(db *v1.DB, kubeConfig *v1.Config) {
	log.Printf("Starting RPC server on port %v", *rpcPort)
	lis, err := net.Listen("tcp", *rpcPort)
	if err != nil {
		log.Fatalf("Failed to start RPC listener: %v", err)
	}

	// Recovery settings
	recoveryFunc = func(p interface{}) (err error) {
		return status.Errorf(codes.Unknown, "panic triggered: %v", p)
	}
	recoveryOpts := []grpc_recovery.Option{
		grpc_recovery.WithRecoveryHandler(recoveryFunc),
	}

	// Logger settings
	stdLogger := log.StandardLogger()
	reportCaller := env.GetEnv("LOGGING_ENABLE_CALLER_TRACE", "false")
	if reportCaller == "true" {
		stdLogger.SetReportCaller(true)
	}
	logEntry := log.NewEntry(stdLogger)

	s := grpc.NewServer(grpc.UnaryInterceptor(
		grpc_middleware.ChainUnaryServer(
			grpc_logrus.UnaryServerInterceptor(logEntry),
			grpc_recovery.UnaryServerInterceptor(recoveryOpts...),
			auth.AuthUnaryInterceptor(kubeConfig, db)),
	), grpc.StreamInterceptor(
		grpc_middleware.ChainStreamServer(
			grpc_logrus.StreamServerInterceptor(logEntry),
			grpc_recovery.StreamServerInterceptor(recoveryOpts...),
			auth.AuthStreamingInterceptor(kubeConfig, db)),
	))
	api.RegisterWorkflowServiceServer(s, server.NewWorkflowServer())
	api.RegisterSecretServiceServer(s, server.NewSecretServer())
	api.RegisterNamespaceServiceServer(s, server.NewNamespaceServer())

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
	allowedHeaders := handlers.AllowedHeaders([]string{"Content-Type", "Authorization"})

	// Allow PUT. Have to include all others as it clears them out.
	allowedMethods := handlers.AllowedMethods([]string{"HEAD", "GET", "POST", "PUT", "DELETE", "PATCH"})

	if err := http.ListenAndServe(*httpPort, wsproxy.WebsocketProxy(
		handlers.CORS(
			handlers.AllowedOriginValidator(ogValidator), allowedHeaders, allowedMethods)(mux),
		wsproxy.WithTokenCookieName("auth-token"),
	)); err != nil {
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
