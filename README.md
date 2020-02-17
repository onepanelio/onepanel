## Database migrations

Install `goose`:
```bash
go get -u github.com/pressly/goose/cmd/goose
```

Note: Up migration are automatically executed when the application is run.

```bash
goose -dir db create <name> sql                 # Create migration in db folder
goose -dir db postgres "${DB_DATASOURCE_NAME}" up    # Migrate the DB to the most recent version available
goose -dir db postgres "${DB_DATASOURCE_NAME}" down  # Roll back the version by 1
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

Then use `go get -u` to download the following packages:

```bash
go get -u github.com/grpc-ecosystem/grpc-gateway/protoc-gen-grpc-gateway
go get -u github.com/grpc-ecosystem/grpc-gateway/protoc-gen-swagger
go get -u github.com/golang/protobuf/protoc-gen-go
```

This will place three binaries in your `$GOBIN`;

* `protoc-gen-grpc-gateway`
* `protoc-gen-swagger`
* `protoc-gen-go`

Make sure that your `$GOBIN` is in your `$PATH`.

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

If you want the Swagger files to have their definitions separated by '.' instead of one big word, use the
following:
```bash
protoc -I/usr/local/include   -Iapi/third_party/googleapis   -Iapi/   api/*.proto   --go_out=plugins=grpc:api   --grpc-gateway_out=logtostderr=true:api   --swagger_out=fqn_for_swagger_name=true,logtostderr=true:api
```
So instead of `apiListSecretsResponse`, it would become `api.ListSecretsResponse`

## Python Client

Install protoc tool for python.

Build the proto files for Python
```bash
python -m grpc_tools.protoc -I/usr/local/include  -Iapi/third_party/googleapis  -Iapi/ api/third_party/googleapis/google/api/*.proto api/third_party/googleapis/google/rpc/*.proto api/*.proto --python_out=. --grpc_python_out=.
```
Run main.go, then run main.py to test the request.

OpenAPI, go to their github for reference.
To generate the python client:
```bash
java -jar openapi-generator-cli.jar generate -i api/secret.swagger.json -g python -o ./pythonopenapi_client/
```

