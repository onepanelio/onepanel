FROM golang:1.15.5 AS builder

WORKDIR /go/src
COPY . .

RUN go get -d -v ./...
RUN go install -v ./...
RUN go get -u github.com/pressly/goose/cmd/goose
RUN go build -o /go/bin/goose ./cmd/goose/goose.go

FROM golang:1.15.5
COPY --from=builder /go/bin/core .
COPY --from=builder /go/src/db ./db
COPY --from=builder /go/bin/goose .
COPY --from=builder /go/src/manifest ./manifest

EXPOSE 8888
EXPOSE 8887

CMD ["./core"]
