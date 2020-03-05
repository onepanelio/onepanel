jq:
	cat api/apidocs.swagger.json \
		| jq 'walk( if type == "object" then with_entries( .key |= sub( "api\\."; "") ) else . end )' \
		| jq 'walk( if type == "string" then gsub( "api\\."; "") else . end )' \
		> api/api.swagger.json
	rm api/apidocs.swagger.json

protoc:
	protoc -I/usr/local/include \
		-Iapi/third_party/ \
 		-Iapi/ \
 		api/*.proto \
 		--go_out=plugins=grpc:api \
 		--grpc-gateway_out=logtostderr=true,allow_delete_body=true:api \
 		--swagger_out=allow_merge=true,fqn_for_swagger_name=true,allow_delete_body=true,logtostderr=true:api

api: protoc jq

python-sdk: openapi-generator
	java -jar openapi-generator-cli.jar generate -p packageName=core.api,projectName=core.api -i api/api.swagger.json -g python -o ./sdks/python

docker-build:
	docker build -t onepanel-core .
	docker tag onepanel-core:latest onepanel/core:1.0.0-beta.1

docker-push:
	docker push onepanel/core:1.0.0-beta.1

docker-all: docker-build docker-push

all: api python-sdk
