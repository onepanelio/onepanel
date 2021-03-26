## Database migrations

### Docker

Note: Up migrations are automatically executed when the application is run.

#### Linux / Mac

```bash
docker run --rm --mount type=bind,source="${PWD}",target=/root onepanel/helper:v1.0.0 goose -dir db/sql create <name> sql  # Create migration in db/sql folder
docker run --rm --mount type=bind,source="${PWD}",target=/root onepanel/helper:v1.0.0 goose -dir db postgres "${DB_DATASOURCE_NAME}" up # Migrate the DB to the most recent version available
docker run --rm --mount type=bind,source="${PWD}",target=/root onepanel/helper:v1.0.0 goose -dir db postgres "${DB_DATASOURCE_NAME}" down # Roll back the version by 1
docker run --rm --mount type=bind,source="${PWD}",target=/root onepanel/helper:v1.0.0 goose help  # See all available commands
```

#### Windows

``bash
docker run --rm --mount type=bind,source="%CD%",target=/root onepanel/helper:v1.0.0 goose -dir db/sql create wow sql  # Create migration in db/sql folder
docker run --rm --mount type=bind,source="%CD%",target=/root onepanel/helper:v1.0.0 goose -dir db postgres "${DB_DATASOURCE_NAME}" up # Migrate the DB to the most recent version available
docker run --rm --mount type=bind,source="%CD%",target=/root onepanel/helper:v1.0.0 goose -dir db postgres "${DB_DATASOURCE_NAME}" down # Roll back the version by 1
docker run --rm --mount type=bind,source="%CD%",target=/root onepanel/helper:v1.0.0 goose help  # See all available commands
``

### Local 

Install `goose`:
```bash
go get -u github.com/pressly/goose/cmd/goose
```

Note: Up migrations are automatically executed when the application is run.

```bash
goose -dir db/sql create <name> sql                   # Create migration in db/sql folder
goose -dir db postgres "${DB_DATASOURCE_NAME}" up     # Migrate the DB to the most recent version available
goose -dir db postgres "${DB_DATASOURCE_NAME}" down   # Roll back the version by 1
goose help                                            # See all available commands
```

## gRPC 

### local installation

Install gRPC:
```bash
go get -u google.golang.org/grpc
```

Download pre-compiled binaries for your platform (protoc-<version>-<platform>.zip) from here: https://github.com/google/protobuf/releases

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

## API code generation

### Docker

Generate Go and Swagger APIs

#### Linux / Mac

```bash
docker run --rm --mount type=bind,source="${PWD}",target=/root onepanel/helper:v1.0.0 make api-internal version=1.0.0
```

#### Windows

```bash
docker run --rm --mount type=bind,source="%CD%",target=/root onepanel/helper:v1.0.0 make api-internal version=1.0.0
```

### Local Installation

Generate Go and Swagger APIs:
```bash
make api-internal version=1.0.0
```

## Code Structure & Organization

### `utils` dir

```shell script
utils/*.go
```

Utils are intended to stand-alone.
- They do not track state
- They are meant to mutate metadata

Do not add onepanel specific code in here.
- Such as "Client"

```shell script
pkg/*.go
```
Code here has to be package friendly.
- Meaning, you can pull the code out into it's own package as needed

That's why you see
```shell script
workspace_template.go
workspace_template_test.go
workspace_template_types.go
```
These can be pulled out into their own package or into a new v2 directory if needed.

You can add
- kubernetes specific operations
- database specific operations
- types

### `cmd` dir
Each source file here is assumed to result in an executable.
- Hence the `package main` at the top of each

Place each source file into it's own folder.
Example source file name: `flush_cache.go`
- Dir structure: `cmd/flush-cache/flush_cache.go`

To avoid errors like this during docker build
```text
# github.com/onepanelio/core/cmd
cmd/goose.go:22:6: main redeclared in this block
        previous declaration at cmd/gen-release-md.go:136:6
github.com/onepanelio/core
```
Caused by
```dockerfile
RUN go install -v ./...
```

