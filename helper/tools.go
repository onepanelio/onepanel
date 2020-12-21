// +build tools

package corehelper

import (
	// this is for finding the package versions
	_ "github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-grpc-gateway"
	// this is for finding the package versions
	_ "github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-openapiv2"
	// this is for finding the package versions
	_ "google.golang.org/grpc/cmd/protoc-gen-go-grpc"
	// this is for finding the package versions
	_ "google.golang.org/protobuf/cmd/protoc-gen-go"
)
