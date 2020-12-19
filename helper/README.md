# Onepanel Core Helper

Helper provides the files to build a Docker image to assist 
with code generation for the onepanel-core project.

In particular, you can use the docker image to generate all of the gRPC related files, as well as create migrations.

## Build

To build, it should be sufficient to run 
```bash
docker build -t onepanel/helper:v1.0.0 .
```

### Updating

Create a new directory somewhere outside of this project.

Copy the `tools.go` file there and change into that directory in your terminal.

Run
```bash
go mod init onepanel-tools
go mod tidy
```

Then, take the `go.mod` and `go.sum` files and copy them back here. 

Then run the # Build steps.