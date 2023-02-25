module github.com/onepanelio/core

go 1.15

require (
	cloud.google.com/go/storage v1.10.0
	github.com/Azure/go-autorest/autorest/adal v0.8.2 // indirect
	github.com/Masterminds/squirrel v1.1.0
	github.com/argoproj/argo v0.0.0-20210112203504-f97bef5d0036
	github.com/argoproj/pkg v0.2.0
	github.com/asaskevich/govalidator v0.0.0-20200428143746-21a406dcc535
	github.com/elazarl/goproxy v0.0.0-20191011121108-aa519ddbe484 // indirect
	github.com/ghodss/yaml v1.0.0
	github.com/golang/protobuf v1.4.3
	github.com/google/uuid v1.1.2
	github.com/gorilla/handlers v1.4.2
	github.com/grpc-ecosystem/go-grpc-middleware v1.2.0
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.0.1
	github.com/hashicorp/go-uuid v1.0.2 // indirect
	github.com/hashicorp/golang-lru v0.5.4 // indirect
	github.com/jmoiron/sqlx v1.2.0
	github.com/lib/pq v1.3.0
	github.com/mattn/go-sqlite3 v2.0.3+incompatible // indirect
	github.com/minio/minio-go/v6 v6.0.45
	github.com/pkg/errors v0.9.1
	github.com/pressly/goose v2.6.0+incompatible
	github.com/sirupsen/logrus v1.6.0
	github.com/stretchr/testify v1.6.1
	github.com/tmc/grpc-websocket-proxy v0.0.0-20200122045848-3419fae592fc
	golang.org/x/net v0.7.0
	golang.org/x/oauth2 v0.0.0-20200902213428-5d25da1a8d43
	google.golang.org/api v0.30.0
	google.golang.org/genproto v0.0.0-20201019141844-1ed22bb0c154
	google.golang.org/grpc v1.33.1
	google.golang.org/protobuf v1.25.0
	gopkg.in/yaml.v2 v2.2.8
	gopkg.in/yaml.v3 v3.0.0-20200615113413-eeeca48fe776
	istio.io/api v0.0.0-20200107183329-ed4b507c54e1
	k8s.io/api v0.18.2
	k8s.io/apimachinery v0.18.2
	k8s.io/client-go v0.18.2
	sigs.k8s.io/yaml v1.2.0
)

replace (
	k8s.io/api => k8s.io/api v0.17.8
	k8s.io/apimachinery => k8s.io/apimachinery v0.17.8
	k8s.io/client-go => k8s.io/client-go v0.17.8
)
