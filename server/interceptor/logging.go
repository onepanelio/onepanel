package interceptor

import (
	"context"

	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

func LoggingUnaryInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
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
