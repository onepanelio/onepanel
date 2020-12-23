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
		-Iapi/proto \
		--go_out ./api/gen --go_opt paths=source_relative \
		--go-grpc_out ./api/gen --go-grpc_opt paths=source_relative \
		--go-grpc_opt paths=source_relative \
		--grpc-gateway_out ./api/gen \
		--grpc-gateway_opt logtostderr=true \
		--grpc-gateway_opt allow_delete_body=true \
	    --grpc-gateway_opt paths=source_relative \
        --grpc-gateway_opt generate_unbound_methods=true \
		--openapiv2_out ./api \
		--openapiv2_opt allow_merge=true \
		--openapiv2_opt fqn_for_openapi_name=true \
		--openapiv2_opt allow_delete_body=true \
		--openapiv2_opt logtostderr=true \
		--openapiv2_opt simple_operation_ids=true \
		api/proto/*.proto

api: init protoc jq

api-docker: init
	docker run --rm --mount type=bind,source="${PWD}",target=/root onepanel/helper:v1.0.0 make api version=$(version)

docker-build:
	docker build -t onepanel-core .
	docker tag onepanel-core:latest onepanel/core:$(COMMIT_HASH)

docker-push:
	docker push onepanel/core:$(COMMIT_HASH)

docker-custom:
	docker build -t onepanel-core .
	docker tag onepanel-core:latest onepanel/core:$(TAG)
	docker push onepanel/core:$(TAG)

docker: docker-build docker-push

run-tests:
	docker run --rm --name test-onepanel-postgres -p 5432:5432 -e POSTGRES_USER=admin -e POSTGRES_PASSWORD=tester -e POSTGRES_DB=onepanel -d  postgres:12.3
	go test github.com/onepanelio/core/pkg -count=1 ||:
	docker stop test-onepanel-postgres