COMMIT_HASH=$(shell git rev-parse --short HEAD)

.PHONY: init

init:
ifndef version
	$(error version is undefined)
endif

jq:
	cat api/apidocs.swagger.json \
		| jq 'walk( if type == "object" then with_entries( .key |= sub( "api\\."; "") ) else . end )' \
		| jq 'walk( if type == "string" then gsub( "api\\."; "") else . end )' \
		| jq '.info.version = "$(version)"' \
		> api/api.swagger.json
	rm api/apidocs.swagger.json

protoc:
	protoc -I/usr/local/include \
		-Iapi/third_party/ \
 		-Iapi/ \
 		api/*.proto \
 		--go_out=plugins=grpc:api \
 		--grpc-gateway_out=logtostderr=true,allow_delete_body=true:api \
 		--swagger_out=allow_merge=true,fqn_for_swagger_name=true,allow_delete_body=true,logtostderr=true,simple_operation_ids=true:api

api: init protoc jq

docker-build:
	docker build -t onepanel-core .
	docker tag onepanel-core:latest onepanel/core:$(COMMIT_HASH)

docker-push:
	docker push onepanel/core:$(COMMIT_HASH)

docker: docker-build docker-push
