## gRPC installation

Install gRPC:
```bash
go get -u google.golang.org/grpc
```

Download pre-compiled binaries for your platform(protoc-<version>-<platform>.zip) from here: https://github.com/google/protobuf/releases

On macOS or Linux:

- Unzip `protoc-<version>-<platform>.zip`
- Move `bin/protoc` to `/usr/local/bin/`
- Move `include/google` to `/usr/local/include`

## gRPC Code generation

Generate Go server and client code:

```bash
protoc -I/usr/local/include \
  -Iapi/third_party/googleapis \
  -Iapi/ \
  api/*.proto \
  --go_out=plugins=grpc:api
```

Generate HTTP reverse-proxy:

```bash
protoc -I/usr/local/include \
  -Iapi/third_party/googleapis \
  -Iapi/ \
  api/*.proto \
  --grpc-gateway_out=logtostderr=true:api
```

Generate Swagger definitions:

```bash
protoc -I/usr/local/include \
  -Iapi/third_party/googleapis \
  -Iapi/ \
  api/*.proto \
  --swagger_out=logtostderr=true:api
```