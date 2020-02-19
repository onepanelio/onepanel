swagger:
	protoc  -I/usr/local/include \
			-Iapi/third_party/googleapis \
			-Iapi/ api/*.proto \
			--go_out=plugins=grpc:api \
			--grpc-gateway_out=logtostderr=true,allow_delete_body=true:api \
			--swagger_out=logtostderr=true,allow_delete_body=true:api
