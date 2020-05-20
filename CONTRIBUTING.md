## Database migrations

Install `goose`:
```bash
go get -u github.com/pressly/goose/cmd/goose
```

Note: Up migrations are automatically executed when the application is run.

```bash
goose -dir db create <name> sql                       # Create migration in db folder
goose -dir db postgres "${DB_DATASOURCE_NAME}" up     # Migrate the DB to the most recent version available
goose -dir db postgres "${DB_DATASOURCE_NAME}" down   # Roll back the version by 1
goose help                                            # See all available commands
```

## gRPC installation

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

Generate Go and Swagger APIs:
```bash
make api
```

## Minikube Debugging and Development

It is possible to access host resources with minikube.
- This means you can run core and core-ui on your machine, and have minikube
execute API calls to your machine.

NOTE:
- Do not use host access with Minikube and VMWare. This has been shown not to work
in our testing.
If you have a work-around, feel free to let us know.

To make this work, some setup is needed.
- Minikube started with driver=virtualbox

Get your Minikube ssh IP
https://minikube.sigs.k8s.io/docs/handbook/host-access/

```shell script
minikube ssh "route -n | grep ^0.0.0.0 | awk '{ print \$2 }'"
```
Example output:
```shell script
10.0.2.2
```

When running core api, add these ENV variables.
```shell script
ONEPANEL_CORE_SERVICE_HOST=10.0.2.2 # IP you just got
ONEPANEL_CORE_SERVICE_PORT=8888 # HTTP Port set in main.go
```

DB Access
- You will need to change the Postgres service from ClusterIP to NodePort

Run
```shell script
minikube service list
```

Look at Postgres, you'll see something like this:
```shell script
$ minikube service list
|----------------------|----------------------------------------|--------------------|--------------------------------|
|      NAMESPACE       |                  NAME                  |    TARGET PORT     |              URL               |
|----------------------|----------------------------------------|--------------------|--------------------------------|
| application-system   | application-controller-manager-service | No node port       |
| default              | kubernetes                             | No node port       |
| kube-system          | kube-dns                               | No node port       |
| kubernetes-dashboard | dashboard-metrics-scraper              | No node port       |
| kubernetes-dashboard | kubernetes-dashboard                   | No node port       |
| onepanel             | onepanel-core                          | http/8888          | http://192.168.99.101:32000    |
|                      |                                        | grpc/8887          | http://192.168.99.101:32001    |
| onepanel             | onepanel-core-ui                       | http/80            | http://192.168.99.101:32002    |
| onepanel             | postgres                               |               5432 | http://192.168.99.101:31975    |
|----------------------|----------------------------------------|--------------------|--------------------------------|
```
Grab `http://192.168.99.101:31975`
Use this in main.go for the following lines:

```shell script
	databaseDataSourceName := fmt.Sprintf("port=31975 host=%v user=%v password=%v dbname=%v sslmode=disable",
		"192.168.99.101", config["databaseUsername"], config["databasePassword"], config["databaseName"])
```
This should connect your developing core to the minikube db.

After this, build main.go and run the executable.
- Or use your IDE equivalent

## Code Structure & Organization

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