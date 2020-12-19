FROM golang:1.15.5 AS builder

WORKDIR /

RUN apt-get update
RUN apt-get install -y --no-install-recommends unzip=6.0-23+deb10u1
RUN curl -sL -o protoc.zip https://github.com/protocolbuffers/protobuf/releases/download/v3.14.0/protoc-3.14.0-linux-x86_64.zip
RUN unzip protoc.zip -d proto
RUN curl -sL -o jq https://github.com/stedolan/jq/releases/download/jq-1.6/jq-linux64
RUN chmod +x jq

FROM golang:1.15.5

WORKDIR /root
COPY ./go.* ./

RUN go get -u github.com/pressly/goose/cmd/goose
RUN go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-grpc-gateway \
    github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-openapiv2 \
    google.golang.org/protobuf/cmd/protoc-gen-go \
    google.golang.org/grpc/cmd/protoc-gen-go-grpc

RUN rm go.mod go.sum

COPY --from=builder /jq /usr/local/bin
COPY --from=builder /proto/bin/protoc /usr/local/bin
COPY --from=builder /proto/include /usr/local/include/

CMD ["/bin/bash"]