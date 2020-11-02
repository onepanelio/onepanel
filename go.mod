module github.com/onepanelio/core

go 1.14

require (
	cloud.google.com/go/storage v1.6.0
	github.com/Azure/go-autorest v14.0.0+incompatible // indirect
	github.com/Azure/go-autorest/autorest/adal v0.8.2 // indirect
	github.com/Masterminds/squirrel v1.1.0
	github.com/argoproj/argo v0.0.0-20201001162359-6f738db0733d
	github.com/argoproj/pkg v0.1.0
	github.com/asaskevich/govalidator v0.0.0-20200428143746-21a406dcc535
	github.com/dgrijalva/jwt-go v3.2.0+incompatible
	github.com/elazarl/goproxy v0.0.0-20191011121108-aa519ddbe484 // indirect
	github.com/evanphx/json-patch v4.5.0+incompatible // indirect
	github.com/ghodss/yaml v1.0.0
	github.com/go-sql-driver/mysql v1.5.0 // indirect
	github.com/golang/protobuf v1.4.2
	github.com/gorilla/handlers v1.4.2
	github.com/grpc-ecosystem/go-grpc-middleware v1.2.0
	github.com/grpc-ecosystem/grpc-gateway v1.14.6
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
	golang.org/x/net v0.0.0-20200602114024-627f9648deb9
	golang.org/x/oauth2 v0.0.0-20200107190931-bf48bf16ab8d
	google.golang.org/api v0.20.0
	google.golang.org/genproto v0.0.0-20200526211855-cb27e3aa2013
	google.golang.org/grpc v1.29.1
	google.golang.org/protobuf v1.23.1-0.20200526195155-81db48ad09cc
	gopkg.in/yaml.v2 v2.2.8
	gopkg.in/yaml.v3 v3.0.0-20200615113413-eeeca48fe776
	istio.io/api v0.0.0-20200107183329-ed4b507c54e1
	k8s.io/api v0.18.2
	k8s.io/apimachinery v0.18.2
	k8s.io/client-go v0.18.2
	sigs.k8s.io/structured-merge-diff v0.0.0-20190525122527-15d366b2352e // indirect
	sigs.k8s.io/yaml v1.2.0
)

replace (
	k8s.io/api => k8s.io/api v0.17.8
	k8s.io/apimachinery => k8s.io/apimachinery v0.17.8
	k8s.io/client-go => k8s.io/client-go v0.17.8
)
