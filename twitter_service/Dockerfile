FROM golang:1.8

WORKDIR /go
COPY . .

RUN go get -d ./...
RUN go build twitter-service.go

EXPOSE 10000

ENTRYPOINT ["/go/twitter-service"]