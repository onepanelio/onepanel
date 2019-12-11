package main

import (
	"context"
	"flag"
	"log"
	"net"
	"net/http"

	"github.com/fsnotify/fsnotify"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/onepanelio/core/api"
	"github.com/onepanelio/core/argo"
	"github.com/onepanelio/core/manager"
	"github.com/onepanelio/core/repository"
	"github.com/onepanelio/core/server"
	"github.com/pressly/goose"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
)

var (
	configPath = flag.String("config", "config", "Path to YAML file containing config")
	rpcPort    = flag.String("rpc-port", ":8887", "RPC Port")
	httpPort   = flag.String("http-port", ":8888", "RPC Port")
)

func main() {
	flag.Parse()

	initConfig()

	db := repository.NewDB(viper.GetString("db.driverName"), viper.GetString("DB_DATASOURCE"))
	if err := goose.Run("status", db.BaseConnection(), "db"); err != nil {
		log.Fatalf("goose up: %v", err)
	}

	argoClient := argo.NewClient(viper.GetString("KUBECONFIG"))

	go startRPCServer(db, argoClient)
	startHTTPProxy()
}

func initConfig() {
	viper.AutomaticEnv()
	viper.SetConfigName("config")
	viper.AddConfigPath(*configPath)

	if err := viper.ReadInConfig(); err != nil {
		log.Fatalf("Fatal error config file: %s", err)
	}
	// Watch for configuration change
	viper.WatchConfig()
	viper.OnConfigChange(func(e fsnotify.Event) {
		// Read in config again
		if err := viper.ReadInConfig(); err != nil {
			log.Fatalf("Fatal error config file: %s", err)
		}
	})
}

func startRPCServer(db *repository.DB, argoClient *argo.Client) {
	resourceManager := manager.NewResourceManager(db, argoClient)

	log.Print("Starting RPC server")
	lis, err := net.Listen("tcp", *rpcPort)
	if err != nil {
		log.Fatalf("Failed to start RPC server: %v", err)
	}

	s := grpc.NewServer(grpc.UnaryInterceptor(loggingInterceptor))
	api.RegisterWorkflowServiceServer(s, server.NewWorkflowServer(resourceManager))

	if err := s.Serve(lis); err != nil {
		log.Fatalf("Failed to serve RPC listener: %v", err)
	}
	log.Print("RPC server started")
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

	err := api.RegisterWorkflowServiceHandlerFromEndpoint(ctx, mux, endpoint, opts)
	if err != nil {
		log.Fatalf("Failed to connect to service: %v", err)
	}

	log.Print("Starting HTTP server")
	if err = http.ListenAndServe(*httpPort, mux); err != nil {
		log.Fatalf("Failed to serve HTTP listener: %v", err)
	}
	log.Print("HTTP server started")
}

func loggingInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
	log.Printf("%v handler started", info.FullMethod)
	resp, err = handler(ctx, req)
	if err != nil {
		log.Printf("%s call failed", info.FullMethod)
		return
	}
	log.Printf("%v handler finished", info.FullMethod)
	return
}
