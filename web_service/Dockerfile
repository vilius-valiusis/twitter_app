FROM golang:latest

WORKDIR /go
COPY . .

RUN go get -d -v ./...
RUN go build web-service.go

EXPOSE 8080

ENTRYPOINT ["/go/web-service"]