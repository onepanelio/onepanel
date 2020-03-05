FROM golang:1.13.8

WORKDIR /go/src

WORKDIR .
COPY . .

RUN go get -d -v ./...
RUN go install -v ./...

ENTRYPOINT /go/bin/core

EXPOSE 8888
EXPOSE 8887