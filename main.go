package main

import (
	"context"
	"flag"
	"fmt"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	corev1 "k8s.io/api/core/v1"
	apiv1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	k8runtime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"
	"net"
	"net/http"

	"github.com/gorilla/handlers"
	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_logrus "github.com/grpc-ecosystem/go-grpc-middleware/logging/logrus"
	grpc_recovery "github.com/grpc-ecosystem/go-grpc-middleware/recovery"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/jmoiron/sqlx"
	"github.com/onepanelio/core/api"
	migrations "github.com/onepanelio/core/db/go"
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

	// stopCh is used to indicate when the RPC server should reload.
	// We do this when the configuration has been changed, so the server has the latest configuration
	stopCh := make(chan struct{})

	go func() {
		kubeConfig := v1.NewConfig()
		client, err := v1.NewClient(kubeConfig, nil, nil)
		if err != nil {
			log.Fatalf("Failed to connect to Kubernetes cluster: %v", err)
		}

		go watchConfigmapChanges(client, "onepanel", stopCh, func(configMap *corev1.ConfigMap) error {
			log.Printf("Configmap changed")
			stopCh <- struct{}{}

			return nil
		})

		for {
			client.ClearSystemConfigCache()
			sysConfig, err := client.GetSystemConfig()
			if err != nil {
				log.Fatalf("Failed to get system config: %v", err)
			}

			dbDriverName, databaseDataSourceName := sysConfig.DatabaseConnection()
			// sqlx.MustConnect will panic when it can't connect to DB. In that case, this whole application will crash.
			// This is okay, as the pod will restart and try connecting to DB again.
			// dbDriverName may be nil, but sqlx will then panic.
			db := sqlx.MustConnect(dbDriverName, databaseDataSourceName)
			goose.SetTableName("goose_db_version")
			if err := goose.Run("up", db.DB, "db/sql"); err != nil {
				log.Fatalf("Failed to run database sql migrations: %v", err)
			}

			goose.SetTableName("goose_db_go_version")
			migrations.Initialize()
			if err := goose.Run("up", db.DB, "db/go"); err != nil {
				log.Fatalf("Failed to run database go migrations: %v", err)
			}

			s := startRPCServer(v1.NewDB(db), kubeConfig, sysConfig, stopCh)

			<-stopCh

			s.Stop()
		}
	}()

	startHTTPProxy()
}

func startRPCServer(db *v1.DB, kubeConfig *v1.Config, sysConfig v1.SystemConfig, stopCh chan struct{}) *grpc.Server {
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
			auth.UnaryInterceptor(kubeConfig, db, sysConfig)),
	), grpc.StreamInterceptor(
		grpc_middleware.ChainStreamServer(
			grpc_logrus.StreamServerInterceptor(logEntry),
			grpc_recovery.StreamServerInterceptor(recoveryOpts...),
			auth.StreamingInterceptor(kubeConfig, db, sysConfig)),
	))
	api.RegisterWorkflowTemplateServiceServer(s, server.NewWorkflowTemplateServer())
	api.RegisterCronWorkflowServiceServer(s, server.NewCronWorkflowServer())
	api.RegisterWorkflowServiceServer(s, server.NewWorkflowServer())
	api.RegisterSecretServiceServer(s, server.NewSecretServer())
	api.RegisterNamespaceServiceServer(s, server.NewNamespaceServer())
	api.RegisterAuthServiceServer(s, server.NewAuthServer())
	api.RegisterLabelServiceServer(s, server.NewLabelServer())
	api.RegisterWorkspaceTemplateServiceServer(s, server.NewWorkspaceTemplateServer())
	api.RegisterWorkspaceServiceServer(s, server.NewWorkspaceServer())
	api.RegisterConfigServiceServer(s, server.NewConfigServer())

	go func() {
		if err := s.Serve(lis); err != nil {
			log.Fatalf("Failed to serve RPC server: %v", err)
		}

		log.Printf("Server finished")
	}()

	return s
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

	registerHandler(api.RegisterWorkflowTemplateServiceHandlerFromEndpoint, ctx, mux, endpoint, opts)
	registerHandler(api.RegisterWorkflowServiceHandlerFromEndpoint, ctx, mux, endpoint, opts)
	registerHandler(api.RegisterCronWorkflowServiceHandlerFromEndpoint, ctx, mux, endpoint, opts)
	registerHandler(api.RegisterSecretServiceHandlerFromEndpoint, ctx, mux, endpoint, opts)
	registerHandler(api.RegisterNamespaceServiceHandlerFromEndpoint, ctx, mux, endpoint, opts)
	registerHandler(api.RegisterAuthServiceHandlerFromEndpoint, ctx, mux, endpoint, opts)
	registerHandler(api.RegisterLabelServiceHandlerFromEndpoint, ctx, mux, endpoint, opts)
	registerHandler(api.RegisterWorkspaceTemplateServiceHandlerFromEndpoint, ctx, mux, endpoint, opts)
	registerHandler(api.RegisterWorkspaceServiceHandlerFromEndpoint, ctx, mux, endpoint, opts)
	registerHandler(api.RegisterConfigServiceHandlerFromEndpoint, ctx, mux, endpoint, opts)

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

// watchConfigmapChanges sets up a listener for configmap changes and calls the onChange function when it happens
func watchConfigmapChanges(client *v1.Client, namespace string, stopCh <-chan struct{}, onChange func(*corev1.ConfigMap) error) {
	restClient := client.CoreV1().RESTClient()
	resource := "configmaps"
	fieldSelector := fields.ParseSelectorOrDie(fmt.Sprintf("metadata.name=%s", "onepanel"))
	listFunc := func(options apiv1.ListOptions) (k8runtime.Object, error) {
		options.FieldSelector = fieldSelector.String()
		req := restClient.Get().
			Namespace(namespace).
			Resource(resource).
			VersionedParams(&options, apiv1.ParameterCodec)
		return req.Do().Get()
	}
	watchFunc := func(options apiv1.ListOptions) (watch.Interface, error) {
		options.Watch = true
		options.FieldSelector = fieldSelector.String()
		req := restClient.Get().
			Namespace(namespace).
			Resource(resource).
			VersionedParams(&options, apiv1.ParameterCodec)
		return req.Watch()
	}
	source := &cache.ListWatch{ListFunc: listFunc, WatchFunc: watchFunc}
	_, controller := cache.NewInformer(
		source,
		&corev1.ConfigMap{},
		0,
		cache.ResourceEventHandlerFuncs{
			UpdateFunc: func(old, new interface{}) {
				oldCM := old.(*corev1.ConfigMap)
				newCM := new.(*corev1.ConfigMap)
				if oldCM.ResourceVersion == newCM.ResourceVersion {
					return
				}
				if newCm, ok := new.(*corev1.ConfigMap); ok {
					log.Infof("Detected ConfigMap update.")
					if err := onChange(newCm); err != nil {
						log.Errorf("Error on calling onChange callback: %v", err)
					}
				}
			},
		})

	// We don't want the watcher to ever stop, so give it a channel that will never be hit.
	neverStopCh := make(chan struct{})
	controller.Run(neverStopCh)
}
