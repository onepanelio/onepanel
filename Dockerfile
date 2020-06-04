FROM golang:1.13.10 AS builder

WORKDIR /go/src
COPY . .

RUN go get -d -v ./...
RUN go install -v ./...
RUN go get -u github.com/pressly/goose/cmd/goose
RUN go build -o /go/bin/goose ./cmd/goose.go

FROM golang:1.13.10
COPY --from=builder /go/bin/core .
COPY --from=builder /go/src/db ./db
COPY --from=builder /go/bin/goose .

EXPOSE 8888
EXPOSE 8887

CMD ["./core"]
