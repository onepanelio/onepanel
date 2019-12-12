## Database migrations

Install `goose`:
```bash
go get -u github.com/pressly/goose/cmd/goose
```

Note: Up migration are automatically executed when the application is run.

```bash
goose -dir db create <name> sql                 # Create migration in db folder
goose -dir db postgres "${DB_DATASOURCE}" up    # Migrate the DB to the most recent version available
goose -dir db postgres "${DB_DATASOURCE}" down  # Roll back the version by 1
goose help                                      # See all available commands
```

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

Generate Go, HTTP reverse-proxy and Swagger files:

```bash
protoc -I/usr/local/include \
  -Iapi/third_party/googleapis \
  -Iapi/ \
  api/*.proto \
  --go_out=plugins=grpc:api \
  --grpc-gateway_out=logtostderr=true:api \
  --swagger_out=logtostderr=true:api
```
