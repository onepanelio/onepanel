## Create gRPC code

```bash
protoc -I api/ api/*.proto --go_out=plugins=grpc:api
```