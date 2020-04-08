FROM golang:1.13.8 AS builder

WORKDIR /go/src
COPY . .

RUN go get -d -v ./...
RUN go install -v ./...


FROM golang:1.13.8
COPY --from=builder /go/bin/core .
COPY --from=builder /go/src/db ./db

EXPOSE 8888
EXPOSE 8887

CMD ["./core"]
